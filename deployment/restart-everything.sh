#!/bin/bash

kubectl rollout restart deploy/oauth2-proxy
kubectl rollout restart deploy/traefik
kubectl rollout restart deploy/dex