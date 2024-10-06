// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

package transportinterceptor

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/yarpc/api/transport"
)

// TestNopUnaryInbound ensures NopUnaryInbound calls the underlying handler without modification.
func TestNopUnaryInbound(t *testing.T) {
	var called bool
	handler := transport.UnaryHandlerFunc(func(ctx context.Context, req *transport.Request, resw transport.ResponseWriter) error {
		called = true
		return nil
	})

	err := transportinterceptor.NopUnaryInbound.Handle(context.Background(), &transport.Request{}, nil, handler)
	assert.NoError(t, err)
	assert.True(t, called)
}

// TestApplyUnaryInbound ensures that UnaryInbound middleware wraps correctly.
func TestApplyUnaryInbound(t *testing.T) {
	var called bool
	handler := transport.UnaryHandlerFunc(func(ctx context.Context, req *transport.Request, resw transport.ResponseWriter) error {
		called = true
		return nil
	})

	middleware := transportinterceptor.UnaryInboundFunc(func(ctx context.Context, req *transport.Request, resw transport.ResponseWriter, h transport.UnaryHandler) error {
		assert.False(t, called)
		return h.Handle(ctx, req, resw)
	})

	wrappedHandler := transportinterceptor.ApplyUnaryInbound(handler, middleware)
	err := wrappedHandler.Handle(context.Background(), &transport.Request{}, nil)
	assert.NoError(t, err)
	assert.True(t, called)
}

// TestNopOnewayInbound ensures NopOnewayInbound calls the underlying handler without modification.
func TestNopOnewayInbound(t *testing.T) {
	var called bool
	handler := transport.OnewayHandlerFunc(func(ctx context.Context, req *transport.Request) error {
		called = true
		return nil
	})

	err := transportinterceptor.NopOnewayInbound.HandleOneway(context.Background(), &transport.Request{}, handler)
	assert.NoError(t, err)
	assert.True(t, called)
}

// TestApplyOnewayInbound ensures that OnewayInbound middleware wraps correctly.
func TestApplyOnewayInbound(t *testing.T) {
	var called bool
	handler := transport.OnewayHandlerFunc(func(ctx context.Context, req *transport.Request) error {
		called = true
		return nil
	})

	middleware := transportinterceptor.OnewayInboundFunc(func(ctx context.Context, req *transport.Request, h transport.OnewayHandler) error {
		assert.False(t, called)
		return h.HandleOneway(ctx, req)
	})

	wrappedHandler := transportinterceptor.ApplyOnewayInbound(handler, middleware)
	err := wrappedHandler.HandleOneway(context.Background(), &transport.Request{})
	assert.NoError(t, err)
	assert.True(t, called)
}

// TestNopStreamInbound ensures NopStreamInbound calls the underlying handler without modification.
func TestNopStreamInbound(t *testing.T) {
	var called bool
	handler := transport.StreamHandlerFunc(func(s *transport.ServerStream) error {
		called = true
		return nil
	})

	err := transportinterceptor.NopStreamInbound.HandleStream(&transport.ServerStream{}, handler)
	assert.NoError(t, err)
	assert.True(t, called)
}

// TestApplyStreamInbound ensures that StreamInbound middleware wraps correctly.
func TestApplyStreamInbound(t *testing.T) {
	var called bool
	handler := transport.StreamHandlerFunc(func(s *transport.ServerStream) error {
		called = true
		return nil
	})

	middleware := transportinterceptor.StreamInboundFunc(func(s *transport.ServerStream, h transport.StreamHandler) error {
		assert.False(t, called)
		return h.HandleStream(s)
	})

	wrappedHandler := transportinterceptor.ApplyStreamInbound(handler, middleware)
	err := wrappedHandler.HandleStream(&transport.ServerStream{})
	assert.NoError(t, err)
	assert.True(t, called)
}
