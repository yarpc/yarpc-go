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

// TestNopUnaryOutbound ensures NopUnaryOutbound calls return nil responses and false for IsRunning.
func TestNopUnaryOutbound(t *testing.T) {
	outbound := NopUnaryOutbound

	resp, err := outbound.Call(context.Background(), &transport.Request{})
	assert.NoError(t, err)
	assert.Nil(t, resp)

	assert.False(t, outbound.IsRunning())
	assert.Nil(t, outbound.Transports())
}

// TestNopOnewayOutbound ensures NopOnewayOutbound calls return nil acks and false for IsRunning.
func TestNopOnewayOutbound(t *testing.T) {
	outbound := NopOnewayOutbound

	ack, err := outbound.CallOneway(context.Background(), &transport.Request{})
	assert.NoError(t, err)
	assert.Nil(t, ack)

	assert.False(t, outbound.IsRunning())
	assert.Nil(t, outbound.Transports())
}

// TestNopStreamOutbound ensures NopStreamOutbound calls return nil responses and false for IsRunning.
func TestNopStreamOutbound(t *testing.T) {
	outbound := NopStreamOutbound

	stream, err := outbound.CallStream(context.Background(), &transport.StreamRequest{})
	assert.NoError(t, err)
	assert.Nil(t, stream)

	assert.False(t, outbound.IsRunning())
	assert.Nil(t, outbound.Transports())
}
