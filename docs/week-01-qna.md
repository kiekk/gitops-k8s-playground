# Week 1 Q&A 정리

> Week 1 실습 중 정리한 개념 복습용 Q&A

---

## Q1. Deployment, ReplicaSet, Pod의 관계는?

세 리소스는 계층적 소유 관계이다.

```
Deployment (배포 전략/업데이트 관리)
  └── ReplicaSet (Pod 수 보장)
        └── Pod (실제 컨테이너 실행 단위)
```

- **Pod**: K8s의 최소 배포 단위. 자가 복구 능력 없음
- **ReplicaSet**: 지정한 수만큼 Pod을 항상 유지하는 컨트롤러. 직접 만들 일은 거의 없음
- **Deployment**: ReplicaSet 위에서 롤링 업데이트, 롤백 등 배포 전략을 관리

이미지 버전을 바꾸면 Deployment가 새 ReplicaSet을 생성하고, 기존 ReplicaSet은 `replicas: 0`으로 유지하여 롤백에 대비한다.

| 방식 | 결과 |
|------|------|
| Pod 직접 생성 | 죽으면 끝. 복구 안 됨 |
| ReplicaSet 직접 생성 | 수는 유지되지만 업데이트/롤백 불가 |
| Deployment 사용 | 수 유지 + 무중단 업데이트 + 롤백 전부 가능 |

---

## Q2. Deployment의 spec.replicas가 곧 ReplicaSet 설정인가?

맞다. Deployment의 `spec.replicas`를 설정하면, Deployment가 생성하는 ReplicaSet에 그 값이 그대로 전달된다. ReplicaSet yaml을 별도로 작성할 필요 없이 Deployment yaml 하나로 Deployment → ReplicaSet → Pod 전체 체인이 만들어진다.

---

## Q3. 하나의 Pod에 여러 컨테이너를 넣을 수 있는가?

가능하다. 같은 Pod 내 컨테이너들은:
- **네트워크 공유** — 서로 `localhost`로 통신
- **볼륨 공유** — 같은 디스크를 마운트 가능
- **함께 스케줄링** — 항상 같은 노드에서 실행

단, 독립적인 서비스를 한 Pod에 넣는 것은 안티패턴이다. 주로 **사이드카 패턴**(메인 앱 + 보조 역할)에서만 사용한다.

| 패턴 | 예시 |
|------|------|
| 로그 수집 | 앱 + Fluentd |
| 프록시 | 앱 + Envoy (서비스 메시) |
| 설정 동기화 | 앱 + config reloader |

독립 서비스(예: API 서버 + DB)는 별도 Pod으로 분리하여 각각 독립적으로 스케일링/배포해야 한다.

---

## Q4. 하나의 Pod에 여러 서비스를 구성하면 통신 라우팅은?

같은 Pod 내 컨테이너들은 네트워크를 공유하므로 **포트로 구분**한다.

```yaml
containers:
  - name: api-server
    ports:
      - containerPort: 8080
  - name: admin-panel
    ports:
      - containerPort: 8081
```

Pod 내부에서는 `localhost:포트`로 통신하고, 외부에서는 Service에서 각 포트를 매핑한다.

하지만 이 구조는 스케일링 불가(API만 늘리고 싶어도 admin까지 같이 늘어남), 배포 결합, 장애 전파 등의 문제가 있으므로 **Pod을 분리하고 Service를 통해 통신**하는 것이 정석이다.

---

## Q5. Service DNS 설정은 어떻게 하나?

별도 설정 필요 없다. Service를 만들면 K8s(CoreDNS)가 자동으로 DNS를 등록한다.

```
<service-name>.<namespace>.svc.cluster.local
```

```bash
# 같은 namespace 안에서는 서비스 이름만으로 충분
curl http://admin-service:8081

# 다른 namespace의 서비스에 접근할 때
curl http://admin-service.other-namespace.svc.cluster.local:8081
```

동작 흐름: Pod 요청 → CoreDNS가 서비스명을 ClusterIP로 해석 → kube-proxy가 실제 Pod IP로 라우팅

이 DNS는 **클러스터 내부 전용**이며 외부에서는 사용할 수 없다. 외부 접근이 필요하면 NodePort, LoadBalancer, Ingress를 사용해야 한다.

---

## Q6. matchLabels와 Pod의 label 관계는?

`selector.matchLabels`는 관리 대상 Pod을 찾는 조건이고, `template.metadata.labels`가 Pod에 실제 부여되는 label이다. 이 둘이 일치해야 Deployment가 Pod을 인식할 수 있다.

```yaml
spec:
  selector:
    matchLabels:
      app: sample-app          # "이 label을 가진 Pod을 관리하겠다"
  template:
    metadata:
      labels:
        app: sample-app        # Pod에 부여되는 실제 label
    spec:
      containers:
        - name: sample-app     # 이건 컨테이너 이름 (label 아님)
```

- `metadata.name`: 리소스의 고유 식별자 (1개)
- `metadata.labels`: key-value 태그 (여러 개 가능, 그룹핑/필터링 용도)

---

## Q7. kind란 무엇인가?

Kubernetes IN Docker. Docker 컨테이너를 K8s 노드로 사용해서 로컬에 K8s 클러스터를 만드는 도구이다.

```
일반 K8s: 물리 서버/VM → kubelet → Pod
kind:     Docker 컨테이너(=가상 노드) → kubelet → Pod(컨테이너 안의 컨테이너)
```

| | kind | minikube | k3d |
|------|------|----------|-----|
| 노드 구현 | Docker 컨테이너 | VM | Docker 컨테이너 |
| 멀티 노드 | O | 제한적 | O |
| K8s 호환성 | 공식 K8s 그대로 | 공식 K8s | k3s (경량판) |
| 주 용도 | CI/CD, 테스트 | 로컬 개발 | 로컬 개발 |

주의사항:
- 로컬 이미지를 `kind load docker-image`로 로드해야 함 (안 하면 ImagePullBackOff)
- LoadBalancer 미지원 (NodePort + extraPortMappings 또는 MetalLB 사용)
- 클러스터 삭제 시 모든 데이터 소멸

---

## 과제 결과 메모

### 과제 1: 스케일링
- 스케일 업 시 Scheduler가 노드의 리소스 여유량을 기반으로 Pod 배치 (균등 분배가 아님)
- 스케일 다운 시 최근에 생성된 Pod이 우선 제거됨
- 균등 분배가 필요하면 `topologySpreadConstraints` 사용

### 과제 2: ConfigMap 변경
- ConfigMap 변경은 Pod에 자동 반영되지 않음
- `envFrom`(환경변수) 방식은 `rollout restart` 필요
- `volume` 마운트(파일) 방식은 kubelet이 주기적으로 갱신 (~1분)
- 실무에서는 Reloader 같은 도구로 자동 restart 트리거

### 과제 3: Pod 장애 복구
- Pod 삭제 시 ReplicaSet이 Desired State를 유지하기 위해 자동으로 새 Pod 생성

### 과제 4: Rolling Update & Rollback
- 이미지 변경 시 자동 롤아웃, `rollout undo`로 이전 버전 롤백 확인
- 롤아웃/롤백 시 Pod이 교체되므로 port-forward 재실행 필요