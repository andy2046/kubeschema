# kubeschema

### makes Kubernetes json schema validation for helm charts easier

### Install

```
helm plugin install https://github.com/andy2046/kubeschema
# or check out to local 
go get github.com/andy2046/kubeschema
```

### Usage

```
helm template ./charts/my-chart | helm kubeschema -f my-chart.yaml

cat ./my-chart.yaml | helm kubeschema -f my-chart.yaml

helm kubeschema my-chart.yaml
```
