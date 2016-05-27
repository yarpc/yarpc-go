// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

package rpc

import (
	"fmt"
	"net/http"

	"github.com/yarpc/yarpc-go"
	"github.com/yarpc/yarpc-go/crossdock-go"
	"github.com/yarpc/yarpc-go/crossdock/behavior/params"
	"github.com/yarpc/yarpc-go/transport"
	ht "github.com/yarpc/yarpc-go/transport/http"
	tch "github.com/yarpc/yarpc-go/transport/tchannel"

	"github.com/uber/tchannel-go"
)

// Create creates an RPC from the given parameters or fails the whole behavior.
func Create(t crossdock.T) yarpc.RPC {
	fatals := crossdock.Fatals(t)

	server := t.Param(params.Server)
	fatals.NotEmpty(server, "server is required")

	var outbound transport.Outbound
	trans := t.Param(params.Transport)
	switch trans {
	case "http":
		// Go HTTP servers have keep-alive enabled by default. If we re-use
		// HTTP clients, the same connection will be used to make requests.
		// This is undesirable during tests because we want to isolate the
		// different test requests. Additionally, keep-alive causes the test
		// server to continue listening on the existing connection for some
		// time after we close the listener.
		cl := &http.Client{Transport: new(http.Transport)}
		outbound = ht.NewOutboundWithClient(fmt.Sprintf("http://%s:8081", server), cl)
	case "tchannel":
		ch, err := tchannel.NewChannel("client", nil)
		fatals.NoError(err, "couldn't create tchannel")
		outbound = tch.NewOutbound(ch, tch.HostPort(server+":8082"))
	default:
		fatals.Fail("", "unknown transport %q", trans)
	}

	return yarpc.New(yarpc.Config{
		Name:      "client",
		Outbounds: transport.Outbounds{"yarpc-test": outbound},
	})
}
