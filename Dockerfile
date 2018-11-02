FROM k8s.gcr.io/debian-base-amd64:0.4.0

COPY bin/karydia /usr/local/bin/karydia

USER 65534:65534

CMD ["karydia"]
