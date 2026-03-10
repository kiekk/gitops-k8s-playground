# GitOps K8s Playground

> GitOps 기반 Progressive Delivery 파이프라인을 구축하며 Kubernetes 인프라 운영 역량을 습득하는 8주 커리큘럼

## 목표

- Kubernetes 핵심 개념부터 실무 운영 패턴까지 단계별 습득
- ArgoCD를 활용한 GitOps 기반 CD 파이프라인 구축
- Argo Rollouts를 활용한 Canary / Blue-Green 배포 및 자동 롤백
- 운영 환경과 유사한 실습 환경을 **최소 비용**으로 구성

## 빠른 시작

```bash
# 1. kind 클러스터 생성
./scripts/setup-kind.sh

# 2. 기본 매니페스트 배포
kubectl apply -k manifests/base

# 3. ArgoCD 설치 (Phase 3부터)
./scripts/install-argocd.sh
```

## Repo 구조

```
gitops-k8s-playground/
├── apps/                          # 샘플 애플리케이션 소스코드
│   └── sample-app/
│       ├── main.go                # Go HTTP 서버 (health, info, stress 엔드포인트)
│       ├── Dockerfile             # 멀티스테이지 빌드
│       └── go.mod
├── manifests/                     # K8s 매니페스트 (ArgoCD가 바라보는 곳)
│   ├── base/                      # Kustomize base
│   │   ├── deployment.yaml
│   │   ├── service.yaml
│   │   ├── configmap.yaml
│   │   ├── secret.yaml
│   │   ├── ingress.yaml
│   │   ├── hpa.yaml
│   │   ├── rollout.yaml           # Argo Rollouts Canary
│   │   ├── rollout-bluegreen.yaml # Argo Rollouts Blue-Green
│   │   ├── analysis-template.yaml # Prometheus 기반 자동 분석
│   │   └── kustomization.yaml
│   └── overlays/
│       ├── dev/                   # dev 환경 (replica 1, debug 로깅)
│       └── prod/                  # prod 환경 (replica 3, warn 로깅)
├── infra/                         # 인프라 구성 (클러스터 레벨)
│   ├── argocd/                    # ArgoCD 설치 & Application 정의
│   ├── argo-rollouts/             # Argo Rollouts 설치
│   ├── prometheus/                # Prometheus + Alertmanager
│   ├── ingress-nginx/             # Ingress Controller
│   └── rbac/                      # Role, RoleBinding, ServiceAccount
├── cluster/                       # 클러스터 생성 설정
│   ├── kind-config.yaml           # 로컬 kind 멀티노드 클러스터
│   └── eks/                       # AWS EKS (Phase 5)
├── scripts/                       # 설치/관리 스크립트
│   ├── setup-kind.sh
│   ├── cleanup-kind.sh
│   ├── install-argocd.sh
│   └── install-argo-rollouts.sh
├── .github/workflows/             # GitHub Actions CI/CD
│   ├── ci.yaml
│   └── update-manifest.yaml
├── docs/                          # 학습 노트
│   ├── architecture.html          # 아키텍처 시각화
│   └── week-01.md                 # Week 1 학습 가이드
└── README.md
```

---

## 비용 최적화 실습 환경 전략

### 단계별 환경 전략

| Phase | 환경 | 비용 | 비고 |
|-------|------|------|------|
| Phase 1-2 (K8s 기초/심화) | 로컬 kind 멀티노드 클러스터 | **무료** | 대부분의 K8s 기능 학습 가능 |
| Phase 3-4 (ArgoCD/Rollouts) | 로컬 kind + MetalLB | **무료** | LoadBalancer 타입도 로컬에서 시뮬레이션 |
| Phase 5 (통합 프로젝트) | AWS EKS | **~$10-15** | 실제 운영 환경 체험 |

### 로컬 환경 (Phase 1~4, 비용 $0)

```yaml
# kind 멀티노드 클러스터 설정 (운영 환경과 유사)
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
nodes:
  - role: control-plane
  - role: worker
  - role: worker
```

- **Docker Desktop** (개인 무료) + **kind**: 멀티노드 클러스터 무료 구성
- **MetalLB**: 로컬에서 LoadBalancer 타입 Service 시뮬레이션
- **Nginx Ingress Controller**: Ingress 트래픽 관리
- **Prometheus (kube-prometheus-stack)**: 메트릭 모니터링
- **GitHub Actions**: Public repo 무료 / Private repo 월 2,000분 무료
- **GitHub Container Registry (ghcr.io)**: 무료 이미지 저장소

> **로컬 환경의 한계**: 실제 클라우드 LoadBalancer, 멀티 AZ, 노드 오토스케일링 등은 체험 불가.
> 하지만 K8s 오브젝트, ArgoCD, Argo Rollouts의 핵심 기능은 100% 동일하게 동작합니다.

### AWS EKS (Phase 5, 세션 단위 운영)

| 리소스 | 비용 | 비고 |
|--------|------|------|
| EKS 컨트롤 플레인 | **$0.10/hr** (~$73/월 상시 운영 시) | 클러스터 존재하는 동안 과금 |
| EC2 워커 노드 (t3.medium On-Demand) | $0.0416/hr | 최소 사양, 2vCPU/4GB |
| EC2 워커 노드 (t3.medium **Spot**) | **~$0.013/hr** | On-Demand 대비 약 70% 절감 |
| ALB (Application Load Balancer) | ~$0.025/hr + 트래픽 | Ingress 연동 시 필요 |
| NAT Gateway | $0.045/hr + 트래픽 | Private subnet 사용 시 |

```bash
# 학습 시작 시 - 클러스터 생성 (~15분)
eksctl create cluster -f cluster/eks/cluster-config.yaml

# 학습 종료 시 - 클러스터 삭제 (~5분)
eksctl delete cluster --name gitops-study-cluster
```

**시나리오별 월 예상 비용** (서울 리전 기준):

| 시나리오 | 컨트롤 플레인 | 노드 (2x t3.medium Spot) | 기타 | 합계 |
|---------|-------------|------------------------|------|------|
| 매일 4시간, 주 5일 | ~$8 | ~$2 | ~$2 | **~$12/월** |
| 주말만 8시간, 주 2일 | ~$6.4 | ~$1.7 | ~$1.5 | **~$10/월** |
| 24/7 상시 운영 (비권장) | $73 | ~$19 | ~$20 | **~$112/월** |

**AWS 비용 절감 필수 팁**:
1. **Spot 인스턴스 필수 사용** - 학습 용도에 안정성 불필요, 70% 절감
2. **학습 끝나면 반드시 `eksctl delete cluster`** - 방치 시 $73+/월 과금
3. **NAT Gateway 회피** - Public subnet으로 구성하면 NAT 비용 $0
4. **ALB 대신 NodePort 또는 port-forward** - Phase 5 통합 실습에서만 ALB 사용
5. **AWS Budgets 알림 설정** - 월 $20 초과 시 알림으로 과금 사고 방지

### 총 비용 요약

```
Phase 1~4 (7주): 로컬 kind 멀티노드 → 비용 $0
Phase 5 (1주):   AWS EKS (세션 단위 운영) → 비용 ~$10-15
──────────────────────────────────────────────────
총 8주 학습 비용: ~$10-15
```

> **중요**: AWS Budgets에서 월 $20 예산 알림을 반드시 설정하세요. 클러스터 삭제를 잊으면 하루 $2.4+가 누적됩니다.

### 무료 도구 전체 목록

| 도구 | 비용 | 용도 |
|------|------|------|
| Docker Desktop | 무료 (개인) | 컨테이너 런타임 |
| kind | 무료 | 로컬 K8s 클러스터 |
| kubectl, helm, kustomize | 무료 | K8s CLI 도구 |
| ArgoCD | 무료 (오픈소스) | GitOps CD |
| Argo Rollouts | 무료 (오픈소스) | Progressive Delivery |
| Prometheus + Grafana | 무료 (오픈소스) | 모니터링 + 알림 |
| GitHub + Actions | 무료 | 소스 관리 + CI |
| ghcr.io | 무료 | 컨테이너 이미지 레지스트리 |
| Lens / k9s | 무료 | K8s GUI/TUI 관리 도구 |

---

## 커리큘럼 개요

### Phase 1: Kubernetes 기초 (2주)

#### [Week 1 - 핵심 개념 & 로컬 클러스터](docs/week-01.md)

- **환경 구성**: kind 멀티노드 클러스터 설치
- **핵심 오브젝트 학습**: Pod, ReplicaSet, Deployment, Service, ConfigMap, Secret, Namespace
- **실습**: 샘플 앱 빌드 & 배포, kubectl 명령어 숙달

#### Week 2 - 네트워킹 & 스토리지

- **네트워킹**: Ingress Controller 설치, Ingress 리소스로 도메인 기반 라우팅
- **스토리지**: PV, PVC 개념 이해, StatefulSet 기본
- **운영 기초**: Resource Requests/Limits, Liveness/Readiness Probe, Rolling Update & Rollback

---

### Phase 2: Kubernetes 심화 (1주)

#### Week 3 - 실무 패턴

- **오토스케일링 (HPA)**: CPU/Memory 기반 자동 스케일링, metrics-server
- **리소스 관리 & 자동 복구**: QoS 클래스, OOMKill, Liveness/Readiness Probe 심화
- **워크로드 관리**: Job, CronJob
- **보안 기초**: RBAC (Role, RoleBinding, ClusterRole), ServiceAccount, NetworkPolicy
- **Helm 패키지 매니저**: chart 구조, values.yaml 커스터마이징

---

### Phase 3: ArgoCD - GitOps 기반 CD (2주)

#### Week 4 - ArgoCD 기초

- **GitOps 개념 이해**: Single Source of Truth, Declarative, Pull-based
- **ArgoCD 설치 & 구성**: Helm 또는 manifest, UI 접속, CLI
- **Application 관리**: Git repo 연동, Sync Policy, Self-Heal, Prune

#### Week 5 - ArgoCD 심화

- **멀티 환경 관리**: Kustomize overlays, ApplicationSet
- **이미지 기반 롤백**: Application History, Git revert vs ArgoCD rollback
- **운영 패턴**: Sync Waves & Hooks, Notification 설정

---

### Phase 4: Argo Rollouts - Progressive Delivery (2주)

#### Week 6 - Argo Rollouts 기초

- **Progressive Delivery 개념**: Canary vs Blue-Green vs Rolling Update
- **Canary 배포**: steps 정의, 단계별 트래픽 비율, Manual Promotion / Abort

#### Week 7 - Argo Rollouts 심화 & 운영 모니터링

- **Blue-Green 배포**: Active/Preview Service, autoPromotion
- **AnalysisTemplate**: Prometheus 메트릭 기반 자동 분석 & 롤백
- **Prometheus & Alertmanager**: 임계치 알림 (CPU 80%, Memory 90%, CrashLoopBackOff)
- **Grafana 대시보드**: CPU/Memory 사용률, HPA 스케일링 이력

---

### Phase 5: 통합 프로젝트 (1주)

#### Week 8 - 전체 파이프라인 구축 (AWS EKS)

##### 배포 패턴 (CI -> CD 연결 방식)

> **핵심**: ArgoCD는 ECR(이미지 레지스트리)을 감시하지 않는다. **Git 매니페스트의 이미지 태그 변경만 감지**한다.

| 패턴 | CI가 하는 일 | ArgoCD 설정 | 사용 환경 |
|------|------------|------------|----------|
| **패턴 1: 완전 자동** | 이미지 빌드 + ECR Push + **매니페스트 이미지 태그 자동 수정 & Git Commit** | Auto Sync | Dev, Staging |
| **패턴 2: PR 승인 게이트** | 이미지 빌드 + ECR Push + **이미지 태그 변경 PR 생성** | Manual Sync 또는 Auto | **Prod** |

```
[패턴 1 - Dev 환경 자동 배포]
코드 Push -> GitHub Actions (Build + ECR Push + manifests/dev 이미지 태그 자동 수정)
     -> ArgoCD Auto Sync -> Argo Rollouts Canary -> 자동 Promotion

[패턴 2 - Prod 환경 승인 후 배포]
코드 Push -> GitHub Actions (Build + ECR Push + manifests/prod 이미지 태그 변경 PR 생성)
     -> PR 리뷰 & 승인 & Merge
     -> ArgoCD Sync (수동 또는 자동) -> Argo Rollouts Canary -> Analysis -> Promotion/Rollback
```

##### 검증 시나리오

- **정상 배포**: 새 버전 push -> 자동 Canary -> Promotion
- **장애 배포**: 에러 발생 버전 push -> 자동 Rollback 확인
- **이미지 롤백**: ArgoCD에서 이전 리비전 선택 -> 특정 이미지 버전으로 롤백
- **오토스케일링**: 부하 생성 -> HPA Scale-out -> 부하 제거 -> Scale-in
- **임계치 알림**: CPU 부하 유발 -> Prometheus Alert -> Slack 알림
- **자동 복구**: Memory Limit 초과 -> OOMKill -> 자동 재시작 -> 정상화

---

## 최종 인프라 아키텍처

> 아키텍처 다이어그램은 [docs/architecture.html](docs/architecture.html) 파일을 브라우저에서 열어 확인하세요.

### 배포 흐름 요약

```
1. 개발자가 코드 수정 후 Git Push
2. GitHub Actions가 Docker 이미지 빌드 -> ECR Push
3. GitHub Actions가 manifest repo의 Image Tag 업데이트 (Dev: 자동 커밋, Prod: PR 생성)
4. ArgoCD가 Git 변경 감지 -> EKS에 Sync
5. Argo Rollouts가 Canary 배포 시작 (20% -> 50% -> 100%)
6. Prometheus 메트릭 기반 AnalysisRun 실행
   - 성공 -> 자동 Promotion (100% 트래픽 전환)
   - 실패 -> 자동 Rollback (이전 버전으로 복귀)
```

---

## 학습 노트

| 주차 | 주제 | 문서 |
|------|------|------|
| Week 1 | 핵심 개념 & 로컬 클러스터 | [docs/week-01.md](docs/week-01.md) |

---

## 추천 학습 리소스

| 주제 | 리소스 |
|------|--------|
| Kubernetes | [Kubernetes 공식 문서](https://kubernetes.io/docs/) |
| Kubernetes | [Kubernetes The Hard Way](https://github.com/kelseyhightower/kubernetes-the-hard-way) (심화) |
| ArgoCD | [ArgoCD 공식 문서](https://argo-cd.readthedocs.io/) |
| Argo Rollouts | [Argo Rollouts 공식 문서](https://argo-rollouts.readthedocs.io/) |
| GitOps | [OpenGitOps](https://opengitops.dev/) |
| 실습 환경 | [kind](https://kind.sigs.k8s.io/), [minikube](https://minikube.sigs.k8s.io/) |

---

## 학습 팁

1. **매 단계마다 Git repo에 코드 커밋** - 학습 과정 자체가 포트폴리오
2. **공식 문서를 1차 자료로** - 블로그는 보조 자료로만 활용
3. **장애 시나리오를 의도적으로 만들어볼 것** - Pod crash, OOM, 네트워크 단절 등
4. **`kubectl explain` 활용** - 리소스 필드를 빠르게 확인
5. **YAML 직접 작성** - 복붙 대신 직접 타이핑하며 구조 체득
