#!/bin/bash
rm frogvision
go build -o frogvision
cp frogvision ~/.jfrog/plugins/
