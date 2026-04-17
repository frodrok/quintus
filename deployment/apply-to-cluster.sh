#!/bin/bash
kubectl apply -f https://raw.githubusercontent.com/traefik/traefik/master/docs/content/reference/dynamic-configuration/kubernetes-crd-definition-v1.yml

kubectl apply -f ./k8s
