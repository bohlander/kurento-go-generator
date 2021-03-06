package kurento

import (
	"fmt"
	"log"
	"reflect"
	"strings"
	"unicode"
	"unicode/utf8"
)

var debug = true

// Debug activate debug information.
func Debug(state bool) {
	debug = state
}

// IMadiaElement implements some basic methods as getConstructorParams or Create().
type IMediaObject interface {

	// Return the constructor parameters
	getConstructorParams(IMediaObject, map[string]interface{}) map[string]interface{}

	// Each media object should be able to create another object
	// Those options are sent to getConstructorParams
	Create(IMediaObject, map[string]interface{})

	// Set ID of the element
	setId(string)

	//Implement Stringer
	String() string

	setParent(IMediaObject)
	addChild(IMediaObject)

	setConnection(*Connection)
}

// Create object "m" with given "options"
func (elem *MediaObject) Create(m IMediaObject, options map[string]interface{}) {
	req := elem.getCreateRequest()
	constparams := m.getConstructorParams(elem, options)
	// TODO params["sessionId"]

	reqparams := map[string]interface{}{
		"type":              getMediaElementType(m),
		"constructorParams": constparams,
	}

	if elem.connection.SessionId != "" {
		reqparams["sessionId"] = elem.connection.SessionId
	}
	req["params"] = reqparams

	if debug {
		log.Printf("request to be sent: %+v\n", req)
	}

	m.setConnection(elem.connection)

	res := <-elem.connection.Request(req)

	if debug {
		log.Println("Oncreate response: ", res)
	}

	if res.Result["value"] != "" {
		elem.addChild(m)
		//m.setParent(elem)
		m.setId(res.Result["value"])
	}
}

type eventHandler func(map[string]interface{})

func (elem *MediaObject) Subscribe(event string, cb eventHandler) float64 {
	// tell the connection about this registered event for this mediaId event combo
	handlerId := elem.connection.Subscribe(event, cb)

	// Make API call to register
	req := elem.getSubscribeRequest()
	reqparams := map[string]interface{}{
		"type":   event,
		"object": elem.String(),
	}
	if elem.connection.SessionId != "" {
		reqparams["sessionId"] = elem.connection.SessionId
	}
	req["params"] = reqparams
	res := <-elem.connection.Request(req)
	if debug {
		log.Println("Subscribe response ", res)
	}

	// pass back the token so can be unregistered
	return handlerId
}

// Implement setConnection that allows element to handle connection
func (elem *MediaObject) setConnection(c *Connection) {
	elem.connection = c
}

// Set parent of current element
// BUG(recursion) a recursion happends while testing, I must find why
func (elem *MediaObject) setParent(m IMediaObject) {
	elem.Parent = m
}

// Append child to the element
func (elem *MediaObject) addChild(m IMediaObject) {
	elem.Childs = append(elem.Childs, m)
}

// setId set object id from a KMS response
func (m *MediaObject) setId(id string) {
	m.Id = id
}

// Build a prepared create request
func (m *MediaObject) getCreateRequest() map[string]interface{} {

	return map[string]interface{}{
		"jsonrpc": "2.0",
		"method":  "create",
		"params":  make(map[string]interface{}),
	}
}

// Build a prepared invoke request
func (m *MediaObject) getInvokeRequest() map[string]interface{} {
	req := m.getCreateRequest()
	req["method"] = "invoke"

	return req
}

func (m *MediaObject) getSubscribeRequest() map[string]interface{} {
	req := m.getCreateRequest()
	req["method"] = "subscribe"

	return req
}

// String implements fmt.Stringer interface, return ID
func (m *MediaObject) String() string {
	return m.Id
}

// Return name of the object
func getMediaElementType(i interface{}) string {
	n := reflect.TypeOf(i).String()
	p := strings.Split(n, ".")
	return p[len(p)-1]
}

func mergeOptions(a, b map[string]interface{}) {
	for key, val := range b {
		a[key] = val
	}
}

func lowerFirst(s string) string {
	if s == "" {
		return ""
	}
	r, n := utf8.DecodeRuneInString(s)
	return string(unicode.ToLower(r)) + s[n:]
}

func setIfNotEmpty(param map[string]interface{}, name string, t interface{}) {

	switch v := t.(type) {
	case string:
		if v != "" {
			param[name] = v
		}
	case int, float64:
		if v != 0 {
			param[name] = v
		}
	case bool:
		if v {
			param[name] = v
		}
	case IMediaObject, fmt.Stringer:
		if v != nil {
			val := fmt.Sprintf("%s", v)
			if val != "" {
				param[name] = val
			}
		}
	case ICustomSerializer:
		param[name] = v.CustomSerialize()
		/*default:
		innerParam := make(map[string]interface{})
		val := reflect.ValueOf(v)
		log.Printf("type %v", val)
		// TODO: look this stuff up somehow? Expose as interface as they do above? Tricky
		innerParam["__module__"] = "kurento"
		innerParam["__type__"] = "IceCandidate"

		for i := 0; i < val.NumField(); i++ {
			fieldName := lowerFirst(val.Type().Field(i).Name)
			//log.Printf("setIfNotEmpty %s %v %v", fieldName, val.Field(i).Interface(), val.Field(i).Type().Name())
			setIfNotEmpty(innerParam, fieldName, val.Field(i).Interface())
		}
		param[name] = innerParam
		log.Printf("Setting %s to %v", name, innerParam)*/
	}
}

type ICustomSerializer interface {
	CustomSerialize() map[string]interface{}
}
