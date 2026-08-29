// Package logging 은 context 기반 로그 속성 전파를 담당하는 횡단 관심사 패키지다.
//
// 컨텍스트 인지 핸들러(ContextHandler)가 context 에 담긴 속성을 로그 레코드에 자동으로 병합하고, WithAttr 헬퍼가 context 에 속성을 주입한다.
// 이 패키지는 표준 slog 의 "핸들러에게 context 병합 책임 위임" 설계를 따른다.
package logging

import (
	"context"
	"log/slog"
	"maps"
	"slices"
)

// contextKey 는 컨텍스트 값의 키로 사용하는 미공개 타입이다.
// 미공개 타입을 사용해 외부 패키지가 동일 키로 컬렉션(충돌)하지 못하게 한다.
type contextKey struct{}

// contextAttrs 는 context 에 보관하는 로그 속성 모음이다.
//
// keys(삽입 순서 보존)와 values(값의 진실 공급원)를 함께 사용한다.
// 값은 항상 map 에서 조회하므로, 중복 키의 값 갱신 시 keys 의 기존 위치는 그대로 유지되고 최신 값만 반영된다.
type contextAttrs struct {
	keys   []string
	values map[string]slog.Value
}

// ctxKey 는 contextAttrs 를 context 에 저장/조회할 때 쓰는 키 인스턴스다.
var ctxKey = contextKey{}

// FromContext 는 컨텍스트에서 로그 속성 슬라이스를 추출한다.
// 속성이 없으면 nil 을 반환한다.
// 반환된 슬라이스는 context 값의 스냅샷 복사본이므로 호출자가 자유롭게 사용해도 컨텍스트에 저장된 값에 영향이 없다.
func FromContext(ctx context.Context) []slog.Attr {
	attrs, ok := ctx.Value(ctxKey).(*contextAttrs)
	if !ok || attrs == nil {
		return nil
	}

	result := make([]slog.Attr, 0, len(attrs.keys))
	for _, k := range attrs.keys {
		result = append(result, slog.Attr{Key: k, Value: attrs.values[k]})
	}
	return result
}

// WithAttr 은 컨텍스트에 key/value 속성을 추가한 새 컨텍스트를 반환한다.
//
// 중복 키 처리 규칙:
//   - 키가 없으면: values 에 키-값을 추가하고 keys 에 키를 추가한다 (삽입 순서 유지).
//   - 키가 있으면: values 의 값만 갱신하고 keys 는 건드리지 않는다 (순서·위치 보존).
//
// 컨텍스트는 스냅샷 의미론을 따른다: keys 는 slices.Clone, values 는 maps.Clone 으로 복사한 뒤 새 컨텍스트에 저장한다.
// 덕분에 부모 컨텍스트의 보관 값이 자식 컨텍스트의 갱신에 오염되지 않고, 동시성 데이터 레이스도 발생하지 않는다.
func WithAttr(ctx context.Context, key string, value any) context.Context {
	attrs := fromContext(ctx)

	if attrs == nil {
		attrs = &contextAttrs{
			keys:   []string{key},
			values: map[string]slog.Value{key: slog.AnyValue(value)},
		}
		return context.WithValue(ctx, ctxKey, attrs)
	}

	cloned := &contextAttrs{
		keys:   slices.Clone(attrs.keys),
		values: maps.Clone(attrs.values),
	}
	if _, exists := cloned.values[key]; !exists {
		cloned.keys = append(cloned.keys, key)
	}
	cloned.values[key] = slog.AnyValue(value)

	return context.WithValue(ctx, ctxKey, cloned)
}

// fromContext 는 contextAttrs 를 컨텍스트에서 조회한다. 없으면 nil 을 반환한다.
func fromContext(ctx context.Context) *contextAttrs {
	attrs, _ := ctx.Value(ctxKey).(*contextAttrs)
	return attrs
}

// ContextHandler 는 context 에 담긴 로그 속성을 레코드에 자동 병합하는 slog.Handler 래퍼다.
//
// 중요: slog.Handler 를 임베드하지 않고 필드(next)로 보유한 뒤 4개 메서드를 전부 재래핑한다(golang/go#71116 방어).
// 임베드로 구현하면 Logger.With() 가 이 래퍼를 우회해 context 속성이 조용히 사라지는 버그가 발생한다.
type ContextHandler struct {
	next slog.Handler
}

// NewContextHandler 는 next 를 감싸는 ContextHandler 를 생성한다.
func NewContextHandler(next slog.Handler) *ContextHandler {
	return &ContextHandler{next: next}
}

// Enabled 는 하위 핸들러의 판단을 그대로 위임한다.
func (h *ContextHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.next.Enabled(ctx, level)
}

// Handle 은 컨텍스트에서 속성을 추출해 레코드에 병합한 뒤 하위 핸들러로 전달한다.
//
// 중복 키 병합 정책: 호출부(record) 우선.
// 컨텍스트 속성 중 레코드에 이미 있는 키는 생략하고, 없는 키만 추가한다. ("명시 > 암시" 원칙)
//
// 레코드는 Clone 으로 복사해 사용하므로 호출부가 보유한 원본 레코드의 공유 attr 버퍼가 변형되지 않는다.
func (h *ContextHandler) Handle(ctx context.Context, r slog.Record) error {
	attrs := FromContext(ctx)
	if len(attrs) == 0 {
		return h.next.Handle(ctx, r)
	}

	cloned := r.Clone()
	existing := make(map[string]struct{}, cloned.NumAttrs())
	cloned.Attrs(func(a slog.Attr) bool {
		existing[a.Key] = struct{}{}
		return true
	})

	for _, a := range attrs {
		if _, dup := existing[a.Key]; dup {
			continue
		}
		existing[a.Key] = struct{}{}
		cloned.AddAttrs(a)
	}

	return h.next.Handle(ctx, cloned)
}

// WithAttrs 는 하위 핸들러에 속성을 추가하고 그 결과를 다시 감싼다.
// 재래핑하지 않으면 Logger.With() 가 ContextHandler 를 우회한다 (#71116).
func (h *ContextHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &ContextHandler{next: h.next.WithAttrs(attrs)}
}

// WithGroup 은 하위 핸들러에 그룹을 추가하고 그 결과를 다시 감싼다.
// 재래핑하지 않으면 Logger.WithGroup() 이 ContextHandler 를 우회한다 (#71116).
func (h *ContextHandler) WithGroup(name string) slog.Handler {
	return &ContextHandler{next: h.next.WithGroup(name)}
}
