To configure the webhook, the equivalent of teh following has been done in the generator code:
```bash
$ oc annotate service sriov-fec-webhook-service service.beta.openshift.io/serving-cert-secret-name=sriov-fec-webhook-service
$ oc annotate apiservice v2.sriovfec.intel.com service.beta.openshift.io/inject-cabundle=true
$ oc annotate crd sriovfecclusterconfigs.sriovfec.intel.com service.beta.openshift.io/inject-cabundle=true
$ oc annotate crd sriovfecnodeconfigs.sriovfec.intel.com service.beta.openshift.io/inject-cabundle=true

```

Modify the deployment by adding
```yaml
      volumes:
      - name: apiservice-cert
        secret:
          defaultMode: 420
          items:
          - key: tls.crt
            path: apiserver.crt
          - key: tls.key
            path: apiserver.key
          secretName: sriov-fec-webhook-service
      - name: webhook-cert
        secret:
          defaultMode: 420
          items:
          - key: tls.crt
            path: tls.crt
          - key: tls.key
            path: tls.key
          secretName: sriov-fec-webhook-service
```
Add to manager container:
```yaml
        volumeMounts:
        - mountPath: /apiserver.local.config/certificates
          name: apiservice-cert
        - mountPath: /tmp/k8s-webhook-server/serving-certs
          name: webhook-cert
```