# Week 1: 핵심 개념 & 로컬 클러스터

> Kubernetes 핵심 오브젝트를 이해하고, kind 클러스터에서 샘플 앱을 빌드/배포하며 kubectl 명령어를 숙달한다.

---

## 사전 준비

### 필수 도구 설치

```bash
# Docker Desktop (macOS)
# https://www.docker.com/products/docker-desktop/ 에서 설치

# kind
brew install kind

# kubectl
brew install kubectl

# (선택) k9s - 터미널 기반 K8s 관리 도구
brew install k9s
```

### 설치 확인

```bash
docker version
kind version
kubectl version --client
```

---

## Step 1: kind 클러스터 생성

### 1.1 클러스터 설정 파일 이해

`cluster/kind-config.yaml`:

```yaml
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
nodes:
  - role: control-plane
    kubeadmConfigPatches:
      - |
        kind: ClusterConfiguration
        apiServer:
          extraArgs:
            authorization-mode: Node,RBAC
    extraPortMappings:
      - containerPort: 80
        hostPort: 80
        protocol: TCP
      - containerPort: 443
        hostPort: 443
        protocol: TCP
  - role: worker
  - role: worker
```

**구성 요소 설명**:
- **control-plane 1개**: API Server, etcd, scheduler, controller-manager 실행
- **worker 2개**: 실제 Pod가 스케줄링되는 노드
- **extraPortMappings**: 호스트의 80/443 포트를 control-plane 컨테이너로 매핑 (Ingress 사용 시 필요)
- **RBAC 활성화**: 인증/인가 실습을 위해 명시적으로 설정

### 1.2 클러스터 생성

```bash
# 스크립트로 한 번에 생성 (metrics-server 포함)
./scripts/setup-kind.sh

# 또는 수동으로 생성
kind create cluster --name gitops-study --config cluster/kind-config.yaml
```

### 1.3 클러스터 확인

```bash
# 클러스터 정보
kubectl cluster-info

# 노드 목록 확인
kubectl get nodes
# NAME                          STATUS   ROLES           AGE   VERSION
# gitops-study-control-plane    Ready    control-plane   1m    v1.29.x
# gitops-study-worker           Ready    <none>          1m    v1.29.x
# gitops-study-worker2          Ready    <none>          1m    v1.29.x

# 노드 상세 정보
kubectl get nodes -o wide

# 시스템 Pod 확인 (kube-system 네임스페이스)
kubectl get pods -n kube-system
```

### 1.4 kubectl 컨텍스트 이해

```bash
# 현재 컨텍스트 확인
kubectl config current-context
# kind-gitops-study

# 모든 컨텍스트 목록
kubectl config get-contexts

# 컨텍스트 전환 (여러 클러스터 사용 시)
kubectl config use-context kind-gitops-study
```

---

## Step 2: 핵심 개념 이해

### 2.1 Kubernetes 아키텍처

```
┌─────────────────────────────────────────────┐
│              Control Plane                   │
│  ┌──────────┐ ┌───────────┐ ┌────────────┐ │
│  │API Server│ │ Scheduler │ │ Controller │ │
│  │          │ │           │ │  Manager   │ │
│  └──────────┘ └───────────┘ └────────────┘ │
│  ┌──────────┐                               │
│  │   etcd   │                               │
│  └──────────┘                               │
└─────────────────────────────────────────────┘
          │
          │ kubelet / kube-proxy
          │
┌─────────────────┐  ┌─────────────────┐
│   Worker Node 1 │  │   Worker Node 2 │
│  ┌───┐ ┌───┐   │  │  ┌───┐ ┌───┐   │
│  │Pod│ │Pod│   │  │  │Pod│ │Pod│   │
│  └───┘ └───┘   │  │  └───┘ └───┘   │
└─────────────────┘  └─────────────────┘
```

**핵심 컴포넌트**:
- **API Server**: 모든 요청의 진입점. `kubectl` 명령이 여기로 전달됨
- **etcd**: 클러스터 상태를 저장하는 분산 key-value 스토어
- **Scheduler**: 새 Pod를 어느 노드에 배치할지 결정
- **Controller Manager**: Desired State와 Current State를 비교하며 조정 (Reconciliation Loop)
- **kubelet**: 각 노드에서 Pod를 실제로 관리하는 에이전트
- **kube-proxy**: 각 노드에서 Service 네트워킹을 처리

### 2.2 핵심 오브젝트 관계

```
Deployment (선언적 관리)
  └── ReplicaSet (복제본 수 보장)
       └── Pod (컨테이너 실행 단위)
            └── Container (실제 프로세스)

Service (네트워크 접근)
  └── Endpoints (Pod IP 목록, label selector로 자동 관리)

ConfigMap → Pod에 환경변수/파일로 마운트
Secret    → Pod에 민감 정보 전달 (base64 인코딩)
```

---

## Step 3: 샘플 앱 빌드

### 3.1 앱 소스 코드 살펴보기

`apps/sample-app/main.go`는 Go로 작성된 간단한 HTTP 서버:

| 엔드포인트 | 용도 |
|-----------|------|
| `GET /` | 앱 버전, 호스트명, 환경 정보 반환 |
| `GET /health` | Liveness Probe 용 (항상 `{"status":"ok"}`) |
| `GET /ready` | Readiness Probe 용 (항상 `{"status":"ready"}`) |
| `GET /info` | 환경변수 출력 (ConfigMap/Secret에서 주입된 값) |
| `GET /stress/cpu` | CPU 부하 생성 (HPA 테스트용, 30초) |
| `GET /stress/memory` | 메모리 할당 (OOMKill 테스트용, ~1GB) |

### 3.2 Docker 이미지 빌드

```bash
cd apps/sample-app

# 이미지 빌드
docker build -t sample-app:1.0.0 .

# 빌드 확인
docker images | grep sample-app
# sample-app   1.0.0   abc123   10 seconds ago   15MB

# 로컬 테스트 (선택)
docker run --rm -p 8080:8080 -e APP_VERSION=1.0.0 sample-app:1.0.0

# 다른 터미널에서 확인
curl http://localhost:8080
# {"app":"sample-app","environment":"development","goVersion":"go1.22.x","hostname":"abc123","version":"1.0.0"}

curl http://localhost:8080/health
# {"status":"ok"}
```

### 3.3 kind 클러스터에 이미지 로드

kind는 로컬 Docker 레지스트리를 사용하지 않으므로, 빌드한 이미지를 클러스터에 직접 로드해야 한다:

```bash
# kind 클러스터에 이미지 로드
kind load docker-image sample-app:1.0.0 --name gitops-study

# 확인 (노드에서 이미지 확인)
docker exec gitops-study-worker crictl images | grep sample-app
```

---

## Step 4: Pod 직접 실행해보기

매니페스트를 적용하기 전에, Pod를 직접 만들어보며 개념을 익힌다.

### 4.1 명령형(Imperative)으로 Pod 생성

```bash
# nginx Pod 생성
kubectl run nginx-test --image=nginx:alpine --port=80

# Pod 상태 확인
kubectl get pods
# NAME         READY   STATUS    RESTARTS   AGE
# nginx-test   1/1     Running   0          10s

# Pod 상세 정보
kubectl describe pod nginx-test

# Pod 로그 확인
kubectl logs nginx-test

# Pod 내부 접속
kubectl exec -it nginx-test -- /bin/sh
# / # hostname
# nginx-test
# / # exit

# Pod 삭제
kubectl delete pod nginx-test
```

### 4.2 선언형(Declarative)으로 Pod 생성

```bash
# 임시 Pod YAML 생성 후 적용
cat <<EOF | kubectl apply -f -
apiVersion: v1
kind: Pod
metadata:
  name: sample-test
  labels:
    app: sample-test
spec:
  containers:
    - name: sample-app
      image: sample-app:1.0.0
      ports:
        - containerPort: 8080
      env:
        - name: APP_VERSION
          value: "1.0.0"
EOF

# Pod 상태 확인
kubectl get pod sample-test

# port-forward로 접속 테스트
kubectl port-forward pod/sample-test 8080:8080
# 다른 터미널에서: curl http://localhost:8080

# 정리
kubectl delete pod sample-test
```

### 4.3 명령형 vs 선언형 비교

| 방식 | 명령 | 특징 |
|------|------|------|
| **명령형** | `kubectl run`, `kubectl create` | 빠른 테스트용. 재현 불가 |
| **선언형** | `kubectl apply -f file.yaml` | GitOps의 기본. YAML을 Git에 저장하여 재현 가능 |

> **원칙**: 이후 모든 실습은 **선언형(Declarative)** 방식을 사용한다. 이것이 GitOps의 핵심이다.

---

## Step 5: Namespace

### 5.1 Namespace 개념

Namespace는 클러스터 내의 가상 분리 단위. 같은 클러스터에서 dev/prod 환경을 분리하거나, 팀별로 리소스를 격리할 때 사용한다.

```bash
# 기본 네임스페이스 확인
kubectl get namespaces
# NAME              STATUS   AGE
# default           Active   10m
# kube-system       Active   10m   (K8s 시스템 컴포넌트)
# kube-public       Active   10m
# kube-node-lease   Active   10m

# dev 네임스페이스 생성
kubectl create namespace dev

# prod 네임스페이스 생성
kubectl create namespace prod

# 네임스페이스 확인
kubectl get ns

# 특정 네임스페이스의 리소스 조회
kubectl get pods -n kube-system
kubectl get all -n dev
```

### 5.2 기본 네임스페이스 설정

```bash
# 매번 -n dev를 입력하기 귀찮다면 기본값 설정
kubectl config set-context --current --namespace=dev

# 확인
kubectl config view --minify | grep namespace

# 다시 default로 복원
kubectl config set-context --current --namespace=default
```

---

## Step 6: 매니페스트 배포

### 6.1 ConfigMap 적용

`manifests/base/configmap.yaml`:

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: sample-app-config
data:
  APP_ENV: dev
  APP_VERSION: 1.0.0
  LOG_LEVEL: info
```

```bash
# ConfigMap 적용
kubectl apply -f manifests/base/configmap.yaml

# 확인
kubectl get configmap sample-app-config
kubectl describe configmap sample-app-config

# YAML로 내용 확인
kubectl get configmap sample-app-config -o yaml
```

### 6.2 Secret 적용

`manifests/base/secret.yaml`:

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: sample-app-secret
type: Opaque
data:
  DB_HOST: bG9jYWxob3N0          # echo -n "localhost" | base64
  DB_PASSWORD: Y2hhbmdlbWUxMjM=  # echo -n "changeme123" | base64
```

```bash
# Secret 적용
kubectl apply -f manifests/base/secret.yaml

# 확인 (값은 base64로 표시됨)
kubectl get secret sample-app-secret -o yaml

# 값 디코딩
kubectl get secret sample-app-secret -o jsonpath='{.data.DB_HOST}' | base64 -d
# localhost

kubectl get secret sample-app-secret -o jsonpath='{.data.DB_PASSWORD}' | base64 -d
# changeme123
```

**base64 인코딩/디코딩 방법**:
```bash
# 인코딩
echo -n "my-value" | base64
# bXktdmFsdWU=

# 디코딩
echo "bXktdmFsdWU=" | base64 -d
# my-value
```

> **주의**: base64는 암호화가 아니다. 프로덕션에서는 Sealed Secrets, Vault, SOPS 등을 사용해야 한다.

### 6.3 Deployment 적용

`manifests/base/deployment.yaml` 핵심 구조:

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: sample-app
spec:
  replicas: 2                    # Pod 2개 유지
  selector:
    matchLabels:
      app: sample-app            # 이 label을 가진 Pod를 관리
  template:
    spec:
      containers:
        - name: sample-app
          image: sample-app:latest
          ports:
            - containerPort: 8080
          envFrom:
            - configMapRef:
                name: sample-app-config   # ConfigMap의 모든 키를 환경변수로 주입
            - secretRef:
                name: sample-app-secret   # Secret의 모든 키를 환경변수로 주입
          resources:
            requests:
              cpu: 100m           # 최소 보장 CPU (0.1 core)
              memory: 128Mi       # 최소 보장 메모리
            limits:
              cpu: 500m           # 최대 사용 CPU (0.5 core)
              memory: 256Mi       # 최대 사용 메모리 (초과 시 OOMKill)
          livenessProbe:          # 실패 시 컨테이너 재시작
            httpGet:
              path: /health
              port: 8080
          readinessProbe:         # 실패 시 Service에서 제외
            httpGet:
              path: /ready
              port: 8080
```

```bash
# Deployment 적용
kubectl apply -f manifests/base/deployment.yaml

# 배포 상태 확인
kubectl get deployment sample-app
# NAME         READY   UP-TO-DATE   AVAILABLE   AGE
# sample-app   2/2     2            2           30s

# Pod 확인
kubectl get pods -l app=sample-app
# NAME                          READY   STATUS    RESTARTS   AGE
# sample-app-6d8f9b7c4d-abc12   1/1     Running   0          30s
# sample-app-6d8f9b7c4d-def34   1/1     Running   0          30s

# Pod 상세 정보 (이벤트, 환경변수, 프로브 설정 등)
kubectl describe pod -l app=sample-app

# 앱 로그 확인
kubectl logs -l app=sample-app --tail=10
```

### 6.4 ReplicaSet 이해

Deployment는 내부적으로 ReplicaSet을 생성하여 Pod 수를 관리한다:

```bash
# ReplicaSet 확인
kubectl get replicaset
# NAME                    DESIRED   CURRENT   READY   AGE
# sample-app-6d8f9b7c4d   2         2         2       1m

# Deployment -> ReplicaSet -> Pod 관계
kubectl get deploy,rs,pod -l app=sample-app
```

### 6.5 Service 적용

`manifests/base/service.yaml`:

```yaml
apiVersion: v1
kind: Service
metadata:
  name: sample-app
spec:
  type: ClusterIP              # 클러스터 내부에서만 접근 가능
  selector:
    app: sample-app            # 이 label의 Pod로 트래픽 전달
  ports:
    - port: 80                 # Service 포트
      targetPort: 8080         # Pod 포트
```

```bash
# Service 적용
kubectl apply -f manifests/base/service.yaml

# 확인
kubectl get service sample-app
# NAME         TYPE        CLUSTER-IP     EXTERNAL-IP   PORT(S)   AGE
# sample-app   ClusterIP   10.96.xxx.xx   <none>        80/TCP    10s

# Endpoints 확인 (Service가 연결된 Pod IP 목록)
kubectl get endpoints sample-app

# port-forward로 접속 테스트
kubectl port-forward service/sample-app 8080:80

# 다른 터미널에서 테스트
curl http://localhost:8080
curl http://localhost:8080/health
curl http://localhost:8080/info
```

### 6.6 Service 유형 비교

| 유형 | 접근 범위 | 사용 시나리오 |
|------|----------|-------------|
| **ClusterIP** (기본) | 클러스터 내부만 | 내부 마이크로서비스 간 통신 |
| **NodePort** | 노드IP:포트로 외부 접근 | 개발/테스트 환경 |
| **LoadBalancer** | 외부 LB 생성 | 클라우드 환경 (EKS, GKE) |
| **ExternalName** | DNS CNAME | 외부 서비스 참조 |

---

## Step 7: Kustomize로 한번에 배포

### 7.1 kustomization.yaml 이해

`manifests/base/kustomization.yaml`:

```yaml
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization

resources:
  - deployment.yaml
  - service.yaml
  - configmap.yaml
  - secret.yaml
  - ingress.yaml
  - hpa.yaml
```

Kustomize는 여러 매니페스트를 묶어서 관리하는 도구. 나중에 overlay(dev/prod)로 환경별 차이를 관리할 때 핵심이 된다.

### 7.2 Kustomize 명령어

```bash
# 적용될 매니페스트 미리보기 (dry-run)
kubectl kustomize manifests/base

# 한번에 적용
kubectl apply -k manifests/base

# 한번에 삭제
kubectl delete -k manifests/base
```

---

## Step 8: kubectl 핵심 명령어 정리

### 8.1 조회 (Read)

```bash
# 리소스 목록
kubectl get pods                          # Pod 목록
kubectl get pods -o wide                  # IP, 노드 등 상세 정보 포함
kubectl get pods -o yaml                  # YAML 형태로 출력
kubectl get pods -l app=sample-app        # 라벨 셀렉터로 필터링
kubectl get pods --all-namespaces         # 모든 네임스페이스
kubectl get all                           # 주요 리소스 한눈에

# 상세 정보
kubectl describe pod <pod-name>           # 이벤트, 상태, 설정 등 상세
kubectl describe deployment sample-app

# 리소스 필드 설명 (공식 문서 대체)
kubectl explain deployment.spec.replicas
kubectl explain pod.spec.containers.livenessProbe
```

### 8.2 로그 & 디버깅

```bash
# 로그 조회
kubectl logs <pod-name>                   # 현재 로그
kubectl logs <pod-name> --previous        # 이전 컨테이너 로그 (crash 후)
kubectl logs -l app=sample-app --tail=20  # 라벨로 여러 Pod 로그
kubectl logs -f <pod-name>                # 실시간 스트리밍

# Pod 내부 접속
kubectl exec -it <pod-name> -- /bin/sh    # 셸 접속
kubectl exec <pod-name> -- env            # 환경변수 확인
kubectl exec <pod-name> -- wget -qO- http://localhost:8080/health  # 앱 테스트
```

### 8.3 스케일링 & 업데이트

```bash
# 수동 스케일링
kubectl scale deployment sample-app --replicas=3
kubectl get pods -l app=sample-app -w     # -w: watch 모드로 실시간 확인

# 다시 2개로 축소
kubectl scale deployment sample-app --replicas=2

# 이미지 업데이트 (Rolling Update 발생)
kubectl set image deployment/sample-app sample-app=sample-app:2.0.0

# 롤아웃 상태 확인
kubectl rollout status deployment/sample-app

# 롤아웃 히스토리
kubectl rollout history deployment/sample-app

# 이전 버전으로 롤백
kubectl rollout undo deployment/sample-app
```

### 8.4 삭제

```bash
# 개별 삭제
kubectl delete pod <pod-name>
kubectl delete deployment sample-app

# 파일 기반 삭제
kubectl delete -f manifests/base/deployment.yaml

# Kustomize 기반 전체 삭제
kubectl delete -k manifests/base

# 라벨로 일괄 삭제
kubectl delete pods -l app=sample-app
```

---

## Step 9: 실습 과제

### 과제 1: 스케일링 동작 확인

```bash
# 1. Pod가 어떤 노드에 배포되는지 확인
kubectl get pods -o wide

# 2. replicas를 5로 늘린 후, 2개 워커 노드에 어떻게 분배되는지 확인
kubectl scale deployment sample-app --replicas=5
kubectl get pods -o wide

# 3. 다시 2로 줄이고 어떤 Pod가 제거되는지 관찰
kubectl scale deployment sample-app --replicas=2
kubectl get pods -w
```

### 과제 2: ConfigMap 변경 후 Pod 재시작

```bash
# 1. ConfigMap의 APP_VERSION을 2.0.0으로 변경
kubectl edit configmap sample-app-config
# 또는 manifests/base/configmap.yaml을 수정 후 kubectl apply -f

# 2. Pod를 재시작하여 새 환경변수 반영
kubectl rollout restart deployment/sample-app

# 3. 새 Pod에서 변경된 값 확인
kubectl exec -it $(kubectl get pod -l app=sample-app -o jsonpath='{.items[0].metadata.name}') -- wget -qO- http://localhost:8080/info
```

> **참고**: ConfigMap 변경은 Pod를 자동으로 재시작하지 않는다. `rollout restart` 또는 이미지 태그 변경이 필요하다.

### 과제 3: Pod 장애 복구 관찰

```bash
# 1. 현재 Pod 목록 확인
kubectl get pods

# 2. Pod 하나를 강제 삭제
kubectl delete pod $(kubectl get pod -l app=sample-app -o jsonpath='{.items[0].metadata.name}')

# 3. Deployment가 자동으로 새 Pod를 생성하는지 관찰
kubectl get pods -w
# -> 삭제된 Pod 대신 새 Pod가 즉시 생성됨 (Desired State 유지)
```

### 과제 4: Rolling Update & Rollback

```bash
# 1. 이미지 v2 빌드 & 로드 (main.go에서 APP_VERSION 변경 후)
docker build -t sample-app:2.0.0 apps/sample-app/
kind load docker-image sample-app:2.0.0 --name gitops-study

# 2. 이미지 업데이트
kubectl set image deployment/sample-app sample-app=sample-app:2.0.0

# 3. 롤아웃 상태 관찰
kubectl rollout status deployment/sample-app

# 4. 새 버전 확인
curl http://localhost:8080  # port-forward 필요

# 5. 문제가 있다면 롤백
kubectl rollout undo deployment/sample-app
kubectl rollout status deployment/sample-app
```

---

## Step 10: 정리

### 리소스 삭제

```bash
# 배포한 리소스 정리
kubectl delete -k manifests/base

# 네임스페이스 삭제
kubectl delete namespace dev
kubectl delete namespace prod
```

### 클러스터 삭제

```bash
./scripts/cleanup-kind.sh
# 또는
kind delete cluster --name gitops-study
```

---

## 체크리스트

학습을 마치면 아래 항목을 확인한다:

- [ ] kind 클러스터 생성/삭제를 할 수 있다
- [ ] Pod, Deployment, ReplicaSet의 관계를 설명할 수 있다
- [ ] `kubectl get`, `describe`, `logs`, `exec` 명령어를 자유롭게 사용할 수 있다
- [ ] Deployment를 통해 Pod 수를 조절(scale)할 수 있다
- [ ] ConfigMap으로 환경변수를 주입하고, 변경사항을 반영할 수 있다
- [ ] Secret을 생성하고 base64 인코딩/디코딩을 할 수 있다
- [ ] Service(ClusterIP)를 생성하고 port-forward로 접속할 수 있다
- [ ] Kustomize로 여러 매니페스트를 한번에 적용/삭제할 수 있다
- [ ] Rolling Update와 Rollback을 수행할 수 있다
- [ ] Pod 삭제 시 Deployment가 자동 복구하는 것을 확인했다
- [ ] 명령형(Imperative)과 선언형(Declarative) 방식의 차이를 이해한다

---

## 다음 단계

[Week 2](week-02.md)에서는:
- **Ingress Controller** 설치 및 도메인 기반 라우팅
- **PV/PVC** 스토리지 실습
- **StatefulSet**으로 상태 유지 워크로드 배포
- **Liveness/Readiness Probe** 동작 검증
