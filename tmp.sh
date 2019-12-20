#!/bin/bash
cd /tmp
tar -xvf \
    ipfs-cluster-ctl_v0.11.0_linux-amd64.tar.gz 
tar -xvf \
    ipfs-cluster-follow_v0.12.0-alpha1_linux-amd64.tar.gz
tar -xvf \
    ipfs-cluster-service_v0.11.0_linux-amd64.tar.gz

cp /tmp/ipfs-cluster-ctl/ipfs-cluster-ctl /bin 
cp /tmp/ipfs-cluster-follow/ipfs-cluster-follow  /bin
cp /tmp/ipfs-cluster-service/ipfs-cluster-service /bin
