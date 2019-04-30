package chromedp

import (
	"context"
	"sync/atomic"
	"time"

	"github.com/mailru/easyjson"

	"github.com/nbzx/cdproto"
	"github.com/nbzx/cdproto/cdp"
	"github.com/nbzx/cdproto/dom"
	"github.com/nbzx/cdproto/page"
	"github.com/nbzx/cdproto/target"
)

// Target manages a Chrome DevTools Protocol target.
type Target struct {
	browser   *Browser
	SessionID target.SessionID
	TargetID  target.ID

	listeners []func(ev interface{})

	waitQueue  chan func() bool
	eventQueue chan *cdproto.Message

	// frames is the set of encountered frames.
	frames map[cdp.FrameID]*cdp.Frame

	// cur is the current top level frame.
	cur *cdp.Frame

	// logging funcs
	logf, errf func(string, ...interface{})

	tick chan time.Time
}

func (t *Target) run(ctx context.Context) {
	// tryWaits runs all wait functions on the current top-level frame, if
	// neither are empty.
	//
	// This function is run after each DOM event, since those are the vast
	// majority that we wait on, and approximately every 5ms. The periodic
	// runs are necessary to wait for events such as a node no longer being
	// visible in Chrome.
	tryWaits := func() {
		n := len(t.waitQueue)
		if n == 0 || t.cur == nil {
			return
		}
		for i := 0; i < n; i++ {
			fn := <-t.waitQueue
			if !fn() {
				// try again later.
				t.waitQueue <- fn
			}
		}
	}

	type eventValue struct {
		method cdproto.MethodType
		value  interface{}
	}
	// syncEventQueue is used to handle events synchronously within Target.
	syncEventQueue := make(chan eventValue, 1024)

	// This goroutine receives events from the browser, calls listeners, and
	// then passes the events onto the main goroutine for the target handler
	// to update itself.
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case msg := <-t.eventQueue:
				ev, err := cdproto.UnmarshalMessage(msg)
				if err != nil {
					if _, ok := err.(cdp.ErrUnknownCommandOrEvent); ok {
						// This is most likely an event received from an older
						// Chrome which a newer cdproto doesn't have, as it is
						// deprecated. Ignore that error.
						continue
					}
					t.errf("could not unmarshal event: %v", err)
					continue
				}
				for _, fn := range t.listeners {
					fn(ev)
				}
				syncEventQueue <- eventValue{msg.Method, ev}
			}
		}
	}()

	for {
		select {
		case <-ctx.Done():
			return
		case ev := <-syncEventQueue:
			switch ev.method.Domain() {
			case "Page":
				t.pageEvent(ev.value)
			case "DOM":
				t.domEvent(ctx, ev.value)
				tryWaits()
			}
		case <-t.tick:
			tryWaits()
		}
	}
}

func (t *Target) Execute(ctx context.Context, method string, params easyjson.Marshaler, res easyjson.Unmarshaler) error {
	paramsMsg := emptyObj
	if params != nil {
		var err error
		if paramsMsg, err = easyjson.Marshal(params); err != nil {
			return err
		}
	}
	innerID := atomic.AddInt64(&t.browser.next, 1)
	msg := &cdproto.Message{
		ID:     innerID,
		Method: cdproto.MethodType(method),
		Params: paramsMsg,
	}
	msgJSON, err := easyjson.Marshal(msg)
	if err != nil {
		return err
	}
	sendParams := target.SendMessageToTarget(string(msgJSON)).
		WithSessionID(t.SessionID)
	sendParamsJSON, _ := easyjson.Marshal(sendParams)

	ch := make(chan *cdproto.Message, 1)
	outerID := atomic.AddInt64(&t.browser.next, 1)
	t.browser.cmdQueue <- cmdJob{
		msg: &cdproto.Message{
			ID:     outerID,
			Method: target.CommandSendMessageToTarget,
			Params: sendParamsJSON,
		},
		// We want to grab the response from the inner message.
		resp:   ch,
		respID: innerID,
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	case msg := <-ch:
		switch {
		case msg == nil:
			return ErrChannelClosed
		case msg.Error != nil:
			return msg.Error
		case res != nil:
			return easyjson.Unmarshal(msg.Result, res)
		}
	}
	return nil
}

// documentUpdated handles the document updated event, retrieving the document
// root for the root frame.
func (t *Target) documentUpdated(ctx context.Context) {
	f := t.cur
	if f == nil{
		return
	}
	f.Lock()
	defer f.Unlock()

	// invalidate nodes
	if f.Root != nil {
		close(f.Root.Invalidated)
	}

	f.Nodes = make(map[cdp.NodeID]*cdp.Node)
	var err error
	f.Root, err = dom.GetDocument().WithPierce(true).Do(cdp.WithExecutor(ctx, t))
	if err == context.Canceled {
		return // TODO: perhaps not necessary, but useful to keep the tests less noisy
	}
	if err != nil {
		t.errf("could not retrieve document root for %s: %v", f.ID, err)
		return
	}
	f.Root.Invalidated = make(chan struct{})
	walk(f.Nodes, f.Root)
}

// emptyObj is an empty JSON object message.
var emptyObj = easyjson.RawMessage([]byte(`{}`))

// pageEvent handles incoming page events.
func (t *Target) pageEvent(ev interface{}) {
	var id cdp.FrameID
	var op frameOp

	switch e := ev.(type) {
	case *page.EventFrameNavigated:
		t.frames[e.Frame.ID] = e.Frame
		if e.Frame.ParentID == "" {
			// This frame is only the new top-level frame if it has
			// no parent.
			t.cur = e.Frame
		}
		return

	case *page.EventFrameAttached:
		id, op = e.FrameID, frameAttached(e.ParentFrameID)

	case *page.EventFrameDetached:
		id, op = e.FrameID, frameDetached

	case *page.EventFrameStartedLoading:
		id, op = e.FrameID, frameStartedLoading

	case *page.EventFrameStoppedLoading:
		id, op = e.FrameID, frameStoppedLoading

		// ignored events
	case *page.EventFrameRequestedNavigation:
		return
	case *page.EventDomContentEventFired:
		return
	case *page.EventLoadEventFired:
		return
	case *page.EventFrameResized:
		return
	case *page.EventLifecycleEvent:
		return
	case *page.EventNavigatedWithinDocument:
		return
	case *page.EventJavascriptDialogOpening:
		return
	case *page.EventJavascriptDialogClosed:
		return

	default:
		t.errf("unhandled page event %T", ev)
		return
	}

	f := t.frames[id]
	if f == nil {
		// This can happen if a frame is attached or starts loading
		// before it's ever navigated to. We won't have all the frame
		// details just yet, but that's okay.
		f = &cdp.Frame{ID: id}
		t.frames[id] = f
	}

	f.Lock()
	op(f)
	f.Unlock()
}

// domEvent handles incoming DOM events.
func (t *Target) domEvent(ctx context.Context, ev interface{}) {
	f := t.cur
	var id cdp.NodeID
	var op nodeOp

	switch e := ev.(type) {
	case *dom.EventDocumentUpdated:
		t.documentUpdated(ctx)
		return

	case *dom.EventSetChildNodes:
		id, op = e.ParentID, setChildNodes(f.Nodes, e.Nodes)

	case *dom.EventAttributeModified:
		id, op = e.NodeID, attributeModified(e.Name, e.Value)

	case *dom.EventAttributeRemoved:
		id, op = e.NodeID, attributeRemoved(e.Name)

	case *dom.EventInlineStyleInvalidated:
		if len(e.NodeIds) == 0 {
			return
		}

		id, op = e.NodeIds[0], inlineStyleInvalidated(e.NodeIds[1:])

	case *dom.EventCharacterDataModified:
		id, op = e.NodeID, characterDataModified(e.CharacterData)

	case *dom.EventChildNodeCountUpdated:
		id, op = e.NodeID, childNodeCountUpdated(e.ChildNodeCount)

	case *dom.EventChildNodeInserted:
		id, op = e.ParentNodeID, childNodeInserted(f.Nodes, e.PreviousNodeID, e.Node)

	case *dom.EventChildNodeRemoved:
		id, op = e.ParentNodeID, childNodeRemoved(f.Nodes, e.NodeID)

	case *dom.EventShadowRootPushed:
		id, op = e.HostID, shadowRootPushed(f.Nodes, e.Root)

	case *dom.EventShadowRootPopped:
		id, op = e.HostID, shadowRootPopped(f.Nodes, e.RootID)

	case *dom.EventPseudoElementAdded:
		id, op = e.ParentID, pseudoElementAdded(f.Nodes, e.PseudoElement)

	case *dom.EventPseudoElementRemoved:
		id, op = e.ParentID, pseudoElementRemoved(f.Nodes, e.PseudoElementID)

	case *dom.EventDistributedNodesUpdated:
		id, op = e.InsertionPointID, distributedNodesUpdated(e.DistributedNodes)

	default:
		t.errf("unhandled node event %T", ev)
		return
	}

	n, ok := f.Nodes[id]
	if !ok {
		// Node ID has been invalidated. Nothing to do.
		return
	}

	f.Lock()
	op(n)
	f.Unlock()
}

type TargetOption func(*Target)
