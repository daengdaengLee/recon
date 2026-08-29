package logging

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"slices"
	"testing"
	"testing/slogtest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- 헬퍼 단위 테스트: WithAttr / FromContext ---

func TestWithAttr_새로운키를_추가한다(t *testing.T) {
	ctx := WithAttr(context.Background(), "request_id", "req-1")

	attrs := FromContext(ctx)
	require.Len(t, attrs, 1)
	assert.Equal(t, "request_id", attrs[0].Key)
	assert.Equal(t, "req-1", attrs[0].Value.Any())
}

func TestWithAttr_중복키는_값만_갱신하고_순서를_유지한다(t *testing.T) {
	ctx := WithAttr(context.Background(), "first", "1")
	ctx = WithAttr(ctx, "second", "2")
	ctx = WithAttr(ctx, "first", "updated")

	attrs := FromContext(ctx)
	require.Len(t, attrs, 2)

	// 삽입 순서가 유지되어야 한다 (first, second).
	assert.Equal(t, "first", attrs[0].Key)
	assert.Equal(t, "updated", attrs[0].Value.Any())
	assert.Equal(t, "second", attrs[1].Key)
	assert.Equal(t, "2", attrs[1].Value.Any())
}

func TestWithAttr_부모컨텍스트는_변하지_않는다(t *testing.T) {
	parent := WithAttr(context.Background(), "user", "alice")
	child := WithAttr(parent, "user", "bob") // 같은 키 갱신
	child = WithAttr(child, "team", "core")  // 새 키 추가

	// 부모는 여전히 원래 값만 보관한다.
	parentAttrs := FromContext(parent)
	require.Len(t, parentAttrs, 1)
	assert.Equal(t, "alice", parentAttrs[0].Value.Any())

	// 자식은 갱신 + 추가가 모두 반영된다.
	childAttrs := FromContext(child)
	require.Len(t, childAttrs, 2)
	assert.Equal(t, "bob", childAttrs[0].Value.Any())
}

func TestWithAttr_속성없는컨텍스트는_nil을_반환한다(t *testing.T) {
	assert.Nil(t, FromContext(context.Background()))
}

func TestWithAttr_다른키타입과_공존한다(t *testing.T) {
	type otherKey struct{}
	ctx := context.WithValue(context.Background(), otherKey{}, "unrelated")

	ctx = WithAttr(ctx, "request_id", "req-1")
	attrs := FromContext(ctx)

	require.Len(t, attrs, 1)
	assert.Equal(t, "request_id", attrs[0].Key)
	assert.Equal(t, "unrelated", ctx.Value(otherKey{}))
}

// --- ContextHandler 단위 테스트 ---

// captureHandler 는 Handle 로 전달된 최종 레코드를 캡처하는 테스트용 핸들러다.
// Logger.With() 는 핸들러를 새 인스턴스로 만들기 때문에, 캡처된 레코드는 모든 인스턴스가 공유하는 포인터 슬라이스에 저장된다.
// WithAttrs 로 전달된 속성은 Handle 시점에 레코드에 병합해 실제 표준 핸들러 동작을 흉내낸다.
type captureHandler struct {
	records *[]slog.Record
	attrs   []slog.Attr
}

func newCaptureHandler() *captureHandler {
	return &captureHandler{records: &[]slog.Record{}}
}

func (h *captureHandler) Enabled(context.Context, slog.Level) bool { return true }
func (h *captureHandler) Handle(_ context.Context, r slog.Record) error {
	r = r.Clone()
	r.AddAttrs(h.attrs...)
	*h.records = append(*h.records, r)
	return nil
}
func (h *captureHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &captureHandler{records: h.records, attrs: append(slices.Clone(h.attrs), attrs...)}
}
func (h *captureHandler) WithGroup(string) slog.Handler {
	return &captureHandler{records: h.records, attrs: h.attrs}
}

// recordAttrs 는 레코드의 속성을 키-값 맵으로 변환한다.
func recordAttrs(t *testing.T, r slog.Record) map[string]any {
	t.Helper()
	got := make(map[string]any)
	r.Attrs(func(a slog.Attr) bool {
		got[a.Key] = a.Value.Any()
		return true
	})
	return got
}

func TestContextHandler_컨텍스트속성을_레코드에_병합한다(t *testing.T) {
	capture := newCaptureHandler()
	logger := slog.New(NewContextHandler(capture))

	ctx := WithAttr(context.Background(), "request_id", "req-1")
	logger.InfoContext(ctx, "hello")

	require.Len(t, *capture.records, 1)
	got := recordAttrs(t, (*capture.records)[0])
	assert.Equal(t, "req-1", got["request_id"])
	assert.Equal(t, "hello", (*capture.records)[0].Message)
}

func TestContextHandler_중복키는_호출부가_우선한다(t *testing.T) {
	capture := newCaptureHandler()
	logger := slog.New(NewContextHandler(capture))

	// 로그 인자(record)에 request_id 가 있고 context 에도 같은 키가 있다.
	ctx := WithAttr(context.Background(), "request_id", "ctx-value")
	logger.InfoContext(ctx, "dup", "request_id", "record-value")

	require.Len(t, *capture.records, 1)
	got := recordAttrs(t, (*capture.records)[0])
	assert.Equal(t, "record-value", got["request_id"], "호출부(record) 값이 유지되어야 한다")
}

func TestContextHandler_With전역속성과_컨텍스트속성이_모두_출력된다(t *testing.T) {
	capture := newCaptureHandler()
	base := slog.New(NewContextHandler(capture))

	// 1. With 로 전역 속성 세팅 → 핸들러 WithAttrs 재래핑 경로.
	logger := base.With("service", "recon")
	// 2. 컨텍스트 속성 세팅.
	ctx := WithAttr(context.Background(), "request_id", "req-1")

	logger.InfoContext(ctx, "both")

	require.Len(t, *capture.records, 1)
	got := recordAttrs(t, (*capture.records)[0])
	// With 속성과 컨텍스트 속성이 함께 출력되어야 한다 (피드백 1 시나리오).
	assert.Equal(t, "recon", got["service"])
	assert.Equal(t, "req-1", got["request_id"])
}

func TestContextHandler_With전역속성과_컨텍스트속성_중복시_record우선(t *testing.T) {
	capture := newCaptureHandler()
	logger := slog.New(NewContextHandler(capture)).With("request_id", "with-value")

	ctx := WithAttr(context.Background(), "request_id", "ctx-value")
	logger.InfoContext(ctx, "dup")

	require.Len(t, *capture.records, 1)
	got := recordAttrs(t, (*capture.records)[0])
	assert.Equal(t, "with-value", got["request_id"], "With 로 부착한 키는 record 쪽이므로 우선해야 한다")
}

// --- slogtest 계약 테스트 ---

func TestContextHandler_slogtest_계약을_준수한다(t *testing.T) {
	var buf bytes.Buffer

	slogtest.Run(t,
		// 각 테스트 케이스마다 새 핸들러를 만든다.
		func(t *testing.T) slog.Handler {
			t.Helper()
			buf.Reset()
			return NewContextHandler(slog.NewJSONHandler(&buf, nil))
		},
		// 케이스 실행 후 단일 로그 레코드의 출력을 파싱한다.
		func(t *testing.T) map[string]any {
			t.Helper()
			return parseLogEntry(t, &buf)
		},
	)
}

// parseLogEntry 는 JSON 핸들러가 쓴 마지막 줄을 단일 맵으로 파싱한다.
// 각 slogtest 케이스는 정확히 한 번의 로그 호출을 하므로 마지막 줄이 그 레코드다.
func parseLogEntry(t *testing.T, buf *bytes.Buffer) map[string]any {
	t.Helper()
	lines := bytes.Split(bytes.TrimSpace(buf.Bytes()), []byte{'\n'})
	if len(lines) == 0 {
		t.Fatalf("로그 출력이 없습니다")
	}
	var m = map[string]any{}
	if err := json.Unmarshal(lines[len(lines)-1], &m); err != nil {
		t.Fatalf("slogtest 출력 파싱 실패: %v", err)
	}
	return m
}
