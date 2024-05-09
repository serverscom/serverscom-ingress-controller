# Kubernetes Ingress Controller Manager for Servers.com
Servers.com provides you with the own Ingress controller built upon the Servers.com HTTP(S) (L7) Load Balancer. It can be used along with the LoadBalancer and NodePort Services. 
The Ingress controller is featured with annotations based on the L7 Load Balancer features.

You can find more details on the Servers.com Ingress controller and peculiarities of usage in the [knowledge base](https://www.servers.com/support/knowledge/kubernetes-clusters/serverscom-ingress-controller).

This is an example how an Ingress object with an annotation may look like:

```
apiVersion: v1
kind: Service
metadata:
  name: example-service
  annotations:
    servers.com/load-balancer-balancing-algorithm: "random_least_connections"
```

[![GitHub Actions status](https://github.com/serverscom/serverscom-ingress-controller/workflows/Test/badge.svg)](https://github.com/serverscom/serverscom-ingress-controller/actions)
