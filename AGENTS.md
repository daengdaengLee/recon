# AGENTS.md

이 파일은 코딩 에이전트를 위한 지시 사항이다.
현재 작업 폴더에서 가장 가까운 AGENTS.md 파일이 가장 높은 우선순위를 가진다.

## 대화 언어

- 사용자와의 대화는 한국어로 한다.
- 요약, 계획, 질문, 작업 보고 등 사용자에게 보여주는 모든 텍스트에 동일하게 적용한다.
- 사용자가 다른 언어로 요청한 경우에만 해당 언어를 사용한다.

## 고유명사 표기

- 고유명사와 기술 용어는 한국어 표현과 원문 표현을 함께 적는다. 형식은 `한국어(원문)`.
- 예: 의존성 주입(Dependency Injection), 지속적 통합(Continuous Integration), 리액트(React)
- 같은 문서나 답변 안에서 같은 용어가 반복되면 처음 한 번만 병기하고 이후에는 한국어 표현만 쓴다.
- 관용적으로 원문 그대로 쓰는 약어(API, HTTP, JSON 등)는 병기하지 않는다.

## 코드 작성 언어

- 식별자(변수, 함수, 클래스, 파일 이름)는 영어로 작성한다.
- 사람에게 설명하기 위한 부분은 한국어로 작성한다.
  - 주석
  - 에러 메시지와 예외 메시지
  - 테스트 이름과 테스트 설명
  - 로그 메시지에서 사람이 읽는 설명 부분
  - 문서 문자열(docstring)

```go
// IsSessionExpired 는 세션이 만료되었는지 확인한다.
func IsSessionExpired(session Session) (bool, error) {
	if session.ExpiresAt.IsZero() {
		return false, errors.New("세션에 만료 시각이 없습니다")
	}
	return session.ExpiresAt.Before(time.Now()), nil
}

func TestIsSessionExpired(t *testing.T) {
	t.Run("만료된 세션이면 true를 반환한다", func(t *testing.T) {
		// ...
	})
}
```
