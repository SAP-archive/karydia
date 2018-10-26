package controller

import (
	"context"
	"fmt"
	"time"

	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
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

	logger *logrus.Logger

	karydiaClientset clientset.Interface

	karydiaInformerFactory        informers.SharedInformerFactory
	karydiaSecurityPoliciesLister listers.KarydiaSecurityPolicyLister
	karydiaSecurityPoliciesSynced cache.InformerSynced

	workqueue workqueue.RateLimitingInterface
}

type Config struct {
	Kubeconfig string
	MasterURL  string

	Logger *logrus.Logger
}

func New(ctx context.Context, config *Config) (*Controller, error) {
	cfg, err := clientcmd.BuildConfigFromFlags(config.MasterURL, config.Kubeconfig)
	if err != nil {
		return nil, fmt.Errorf("error building kubeconfig: %v", err)
	}

	karydiaClientset, err := clientset.NewForConfig(cfg)
	if err != nil {
		return nil, fmt.Errorf("error building karydia clientset: %v", err)
	}

	karydiaInformerFactory := informers.NewSharedInformerFactory(karydiaClientset, resyncInterval)

	karydiaSecurityPolicyInformer := karydiaInformerFactory.Karydia().V1alpha1().KarydiaSecurityPolicies()

	controller := &Controller{
		ctx: ctx,

		karydiaClientset: karydiaClientset,

		karydiaInformerFactory:        karydiaInformerFactory,
		karydiaSecurityPoliciesLister: karydiaSecurityPolicyInformer.Lister(),
		karydiaSecurityPoliciesSynced: karydiaSecurityPolicyInformer.Informer().HasSynced,

		workqueue: workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "KarydiaSecurityPolicies"),
	}

	if config.Logger == nil {
		controller.logger = logrus.New()
		controller.logger.Level = logrus.InfoLevel
	} else {
		// convenience
		controller.logger = config.Logger
	}

	karydiaSecurityPolicyInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: controller.enqueueKarydiaSecurityPolicy,
		UpdateFunc: func(old, new interface{}) {
			controller.enqueueKarydiaSecurityPolicy(new)
		},
	})

	return controller, nil
}

func (c *Controller) KarydiaInformerFactory() informers.SharedInformerFactory {
	return c.karydiaInformerFactory
}

func (c *Controller) Run(threadiness int) error {
	defer runtime.HandleCrash()
	defer c.workqueue.ShutDown()

	go c.karydiaInformerFactory.Start(c.ctx.Done())

	if ok := cache.WaitForCacheSync(c.ctx.Done(), c.karydiaSecurityPoliciesSynced); !ok {
		return fmt.Errorf("failed to wait for caches to sync")
	}

	for i := 0; i < threadiness; i++ {
		go wait.Until(c.runWorker, time.Second, c.ctx.Done())
	}

	<-c.ctx.Done()

	return nil
}

func (c *Controller) enqueueKarydiaSecurityPolicy(obj interface{}) {
	var key string
	var err error
	if key, err = cache.MetaNamespaceKeyFunc(obj); err != nil {
		runtime.HandleError(err)
		return
	}
	c.logger.Infof("Enqueuing %q", key)
	c.workqueue.AddRateLimited(key)
}

func (c *Controller) runWorker() {
	for c.processNextWorkItem() {
	}
}

func (c *Controller) processNextWorkItem() bool {
	obj, shutdown := c.workqueue.Get()

	if shutdown {
		return false
	}

	err := func(obj interface{}) error {
		defer c.workqueue.Done(obj)

		var key string
		var ok bool
		if key, ok = obj.(string); !ok {
			// invalid obj in queue, drop it
			c.workqueue.Forget(obj)
			runtime.HandleError(fmt.Errorf("expected string in workqueue but got %#v", obj))
			return nil
		}

		if err := c.syncHandler(key); err != nil {
			return fmt.Errorf("error syncing %q: %v", key, err)
		}

		c.workqueue.Forget(obj)
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
