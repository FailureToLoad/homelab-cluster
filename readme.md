# Homelab Cluster

This repo came about by following [this very informative blog series](https://rcwz.pl/2025-10-04-installing-talos-on-raspberry-pi-5/) by [artuross](https://github.com/artuross). His guide focuses on vendoring manifests for a much higher level of confidence and control when managing a home cluster. Vendoring also sidesteps a lot of sync issues you'll encounter in argo when using direct helm charts.  

My implementation is much lazier. I just want to run a very slim, lightweight repo that gets me a running cluster. I also ran into some annoyances with ArgoCD so I ended up moving to Flux. You lose the very nice visualizations in ArgoCD but for me [k9s](https://k9scli.io/) does the job when I need to know whats going on.  

## Using This Repo

This repo can be viewed more as a template than running source code. Follow the [setup guide](https://github.com/FailureToLoad/homelab-cluster/wiki/Setup) for a walkthrough in standing up your own cluster.  

If you run into any problems, please post an issue!
