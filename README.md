# OpenElect
Host your own elections with OpenElect!

(to be completed)

create configmap from .env file:
```kubectl create configmap openelect-config --from-env-file=.env```

## Deployment (i think)
build:
```docker build -t openelect:latest .```
```docker build -f tailwind.Dockerfile -t tailwind:latest .```
deploy:
```kubectl apply -f openelect/```