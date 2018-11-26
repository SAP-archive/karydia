package controller

import (
	"context"
	"fmt"
	"time"

	"github.com/sirupsen/logrus"
	networkingv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	kubeinformers "k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	kubelistersv1 "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/workqueue"

	clientset "github.com/kinvolk/karydia/pkg/client/clientset/versioned"
	informers "github.com/kinvolk/karydia/pkg/client/informers/externalversions"
	listers "github.com/kinvolk/karydia/pkg/client/listers/karydia/v1alpha1"
)

const resyncInterval = 30 * time.Second

type Controller struct {
	ctx context.Context

	defaultNetworkPolicy         *networkingv1.NetworkPolicy
	defaultNetworkPolicyExcludes []string

	logger *logrus.Logger

	kubeClientset    *kubernetes.Clientset
	karydiaClientset clientset.Interface

	kubeInformerFactory kubeinformers.SharedInformerFactory
	namespacesLister    kubelistersv1.NamespaceLister
	namespacesSynced    cache.InformerSynced

	namespacesWorkqueue workqueue.RateLimitingInterface

	karydiaInformerFactory        informers.SharedInformerFactory
	karydiaSecurityPoliciesLister listers.KarydiaSecurityPolicyLister
	karydiaSecurityPoliciesSynced cache.InformerSynced

	karydiaSecurityPoliciesWorkqueue workqueue.RateLimitingInterface
}

type Config struct {
	Kubeconfig string
	MasterURL  string

	Logger *logrus.Logger

	DefaultNetworkPolicy         *networkingv1.NetworkPolicy
	DefaultNetworkPolicyExcludes []string
}

func New(ctx context.Context, config *Config) (*Controller, error) {
	cfg, err := clientcmd.BuildConfigFromFlags(config.MasterURL, config.Kubeconfig)
	if err != nil {
		return nil, fmt.Errorf("error building kubeconfig: %v", err)
	}

	kubeClientset, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return nil, fmt.Errorf("error building kubernetes clientset: %v", err)
	}

	karydiaClientset, err := clientset.NewForConfig(cfg)
	if err != nil {
		return nil, fmt.Errorf("error building karydia clientset: %v", err)
	}

	kubeInformerFactory := kubeinformers.NewSharedInformerFactory(kubeClientset, resyncInterval)
	karydiaInformerFactory := informers.NewSharedInformerFactory(karydiaClientset, resyncInterval)

	namespaceInformer := kubeInformerFactory.Core().V1().Namespaces()

	karydiaSecurityPolicyInformer := karydiaInformerFactory.Karydia().V1alpha1().KarydiaSecurityPolicies()

	controller := &Controller{
		ctx: ctx,

		defaultNetworkPolicy:         config.DefaultNetworkPolicy,
		defaultNetworkPolicyExcludes: config.DefaultNetworkPolicyExcludes,

		kubeClientset:    kubeClientset,
		karydiaClientset: karydiaClientset,

		kubeInformerFactory: kubeInformerFactory,
		namespacesLister:    namespaceInformer.Lister(),
		namespacesSynced:    namespaceInformer.Informer().HasSynced,

		namespacesWorkqueue: workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "Namespaces"),

		karydiaInformerFactory:        karydiaInformerFactory,
		karydiaSecurityPoliciesLister: karydiaSecurityPolicyInformer.Lister(),
		karydiaSecurityPoliciesSynced: karydiaSecurityPolicyInformer.Informer().HasSynced,

		karydiaSecurityPoliciesWorkqueue: workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "KarydiaSecurityPolicies"),
	}

	if config.Logger == nil {
		controller.logger = logrus.New()
		controller.logger.Level = logrus.InfoLevel
	} else {
		// convenience
		controller.logger = config.Logger
	}

	namespaceInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: controller.handleNamespaceAdd,
	})

	karydiaSecurityPolicyInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: controller.enqueueKarydiaSecurityPolicy,
		UpdateFunc: func(old, new interface{}) {
			controller.enqueueKarydiaSecurityPolicy(new)
		},
	})

	return controller, nil
}

func (c *Controller) KubeInformerFactory() kubeinformers.SharedInformerFactory {
	return c.kubeInformerFactory
}

func (c *Controller) KarydiaInformerFactory() informers.SharedInformerFactory {
	return c.karydiaInformerFactory
}

func (c *Controller) Run(threadiness int) error {
	defer runtime.HandleCrash()
	defer c.karydiaSecurityPoliciesWorkqueue.ShutDown()
	defer c.namespacesWorkqueue.ShutDown()

	go c.karydiaInformerFactory.Start(c.ctx.Done())
	go c.kubeInformerFactory.Start(c.ctx.Done())

	if ok := cache.WaitForCacheSync(c.ctx.Done(), c.karydiaSecurityPoliciesSynced, c.namespacesSynced); !ok {
		return fmt.Errorf("failed to wait for caches to sync")
	}

	for i := 0; i < threadiness; i++ {
		go wait.Until(c.runKSPWorker, time.Second, c.ctx.Done())
		go wait.Until(c.runNamespacesWorker, time.Second, c.ctx.Done())
	}

	<-c.ctx.Done()

	return nil
}

func (c *Controller) handleNamespaceAdd(obj interface{}) {
	var key string
	var err error
	if key, err = cache.MetaNamespaceKeyFunc(obj); err != nil {
		runtime.HandleError(err)
		return
	}
	c.logger.Infof("Enqueuing namespace %q", key)
	c.namespacesWorkqueue.AddRateLimited(key)
}

func (c *Controller) enqueueKarydiaSecurityPolicy(obj interface{}) {
	var key string
	var err error
	if key, err = cache.MetaNamespaceKeyFunc(obj); err != nil {
		runtime.HandleError(err)
		return
	}
	c.logger.Infof("Enqueuing karydia security policy %q", key)
	c.karydiaSecurityPoliciesWorkqueue.AddRateLimited(key)
}

func (c *Controller) runKSPWorker() {
	for c.processNextWorkItem() {
	}
}

func (c *Controller) processNextWorkItem() bool {
	obj, shutdown := c.karydiaSecurityPoliciesWorkqueue.Get()

	if shutdown {
		return false
	}

	err := func(obj interface{}) error {
		defer c.karydiaSecurityPoliciesWorkqueue.Done(obj)

		var key string
		var ok bool
		if key, ok = obj.(string); !ok {
			// invalid obj in queue, drop it
			c.karydiaSecurityPoliciesWorkqueue.Forget(obj)
			runtime.HandleError(fmt.Errorf("expected string in workqueue but got %#v", obj))
			return nil
		}

		if err := c.syncHandler(key); err != nil {
			return fmt.Errorf("error syncing %q: %v", key, err)
		}

		c.karydiaSecurityPoliciesWorkqueue.Forget(obj)
		c.logger.Infof("Successfully synced %q", key)
		return nil
	}(obj)

	if err != nil {
		runtime.HandleError(err)
	}

	return true
}

func (c *Controller) syncHandler(key string) error {
	return nil
}

func (c *Controller) runNamespacesWorker() {
	for c.processNextNamespaceItem() {
	}
}

func (c *Controller) processNextNamespaceItem() bool {
	obj, shutdown := c.namespacesWorkqueue.Get()

	if shutdown {
		return false
	}

	err := func(obj interface{}) error {
		defer c.namespacesWorkqueue.Done(obj)

		var key string
		var ok bool
		if key, ok = obj.(string); !ok {
			// invalid obj in queue, drop it
			c.namespacesWorkqueue.Forget(obj)
			runtime.HandleError(fmt.Errorf("expected string in workqueue but got %#v", obj))
			return nil
		}
		for _, ns := range c.defaultNetworkPolicyExcludes {
			if key == ns {
				c.logger.Infof("Not creating default network policy in %q - namespace excluded", key)
				c.namespacesWorkqueue.Forget(obj)
				return nil
			}
		}

		if err := c.createDefaultNetworkPolicy(key); err != nil {
			c.logger.Errorf("failed to create default network policy: %v", err)
			return fmt.Errorf("error syncing %q: %v", key, err)
		}

		c.namespacesWorkqueue.Forget(obj)
		c.logger.Infof("Successfully synced %q", key)
		return nil
	}(obj)

	if err != nil {
		runtime.HandleError(err)
	}

	return true
}

func (c *Controller) createDefaultNetworkPolicy(namespace string) error {
	if c.defaultNetworkPolicy == nil {
		c.logger.Warn("createDefaultNetworkPolicy called but no policy set")
		return nil
	}
	policyCopy := c.defaultNetworkPolicy.DeepCopy()
	policyCopy.ObjectMeta.Namespace = namespace
	if _, err := c.kubeClientset.NetworkingV1().NetworkPolicies(namespace).Create(policyCopy); err != nil {
		return err
	}
	return nil

}
