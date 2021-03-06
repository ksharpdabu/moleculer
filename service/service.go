package service

import (
	"errors"
	"fmt"
	"strings"

	"github.com/moleculer-go/moleculer"
	log "github.com/sirupsen/logrus"
)

type Action struct {
	name     string
	fullname string
	handler  moleculer.ActionHandler
	params   moleculer.ActionSchema
}

type Event struct {
	name        string
	serviceName string
	group       string
	handler     moleculer.EventHandler
}

func (event *Event) Handler() moleculer.EventHandler {
	return event.handler
}

func (event *Event) Name() string {
	return event.name
}

func (event *Event) ServiceName() string {
	return event.serviceName
}

func (event *Event) Group() string {
	return event.group
}

type Service struct {
	nodeID       string
	fullname     string
	name         string
	version      string
	dependencies []string
	settings     map[string]interface{}
	metadata     map[string]interface{}
	actions      []Action
	events       []Event
	created      moleculer.CreatedFunc
	started      moleculer.LifecycleFunc
	stopped      moleculer.LifecycleFunc
	schema       *moleculer.Service
	logger       *log.Entry
}

func (service *Service) Schema() *moleculer.Service {
	return service.schema
}

func (service *Service) NodeID() string {
	return service.nodeID
}

func (service *Service) Settings() map[string]interface{} {
	return service.settings
}

func (service *Service) SetNodeID(nodeID string) {
	service.nodeID = nodeID
}

func (service *Service) Dependencies() []string {
	return service.dependencies
}

func (serviceAction *Action) Handler() moleculer.ActionHandler {
	return serviceAction.handler
}

func (serviceAction *Action) Name() string {
	return serviceAction.name
}

func (serviceAction *Action) FullName() string {
	return serviceAction.fullname
}

func (service *Service) Name() string {
	return service.name
}

func (service *Service) FullName() string {
	return service.fullname
}

func (service *Service) Version() string {
	return service.version
}

func (service *Service) Actions() []Action {
	return service.actions
}

func (service *Service) Summary() map[string]string {
	return map[string]string{
		"name":    service.name,
		"version": service.version,
		"nodeID":  service.nodeID,
	}
}

func (service *Service) Events() []Event {
	return service.events
}

func findAction(name string, actions []moleculer.Action) bool {
	for _, a := range actions {

		if a.Name == name {
			return true
		}

	}
	return false
}

// extendActions merges the actions from the base service with the mixin schema.
func extendActions(service moleculer.Service, mixin *moleculer.Mixin) moleculer.Service {
	for _, ma := range mixin.Actions {
		if !findAction(ma.Name, service.Actions) {
			service.Actions = append(service.Actions, ma)
		}
	}
	return service
}

func mergeDependencies(service moleculer.Service, mixin *moleculer.Mixin) moleculer.Service {
	list := []string{}
	for _, item := range mixin.Dependencies {
		list = append(list, item)
	}
	for _, item := range service.Dependencies {
		list = append(list, item)
	}
	service.Dependencies = list
	return service
}

func concatenateEvents(service moleculer.Service, mixin *moleculer.Mixin) moleculer.Service {
	for _, mixinEvent := range mixin.Events {
		for _, serviceEvent := range service.Events {
			if serviceEvent.Name != mixinEvent.Name {
				service.Events = append(service.Events, mixinEvent)
			}
		}
	}
	return service
}

func MergeSettings(settings ...map[string]interface{}) map[string]interface{} {
	result := map[string]interface{}{}
	for _, set := range settings {
		if set != nil {
			for key, value := range set {
				result[key] = value
			}
		}
	}
	return result
}

func extendSettings(service moleculer.Service, mixin *moleculer.Mixin) moleculer.Service {
	service.Settings = MergeSettings(mixin.Settings, service.Settings)
	return service
}

func extendMetadata(service moleculer.Service, mixin *moleculer.Mixin) moleculer.Service {
	service.Metadata = MergeSettings(mixin.Metadata, service.Metadata)
	return service
}

func extendHooks(service moleculer.Service, mixin *moleculer.Mixin) moleculer.Service {
	service.Hooks = MergeSettings(mixin.Hooks, service.Hooks)
	return service
}

// chainCreated chain the Created hook of services and mixins
func chainCreated(service moleculer.Service, mixin *moleculer.Mixin) moleculer.Service {
	if mixin.Created != nil {
		svcHook := service.Created
		service.Created = func(svc moleculer.Service, log *log.Entry) {
			if svcHook != nil {
				svcHook(svc, log)
			}
			mixin.Created(svc, log)
		}
	}
	return service
}

// chainStarted chain the Started hook of services and mixins
func chainStarted(service moleculer.Service, mixin *moleculer.Mixin) moleculer.Service {
	if mixin.Started != nil {
		svcHook := service.Started
		service.Started = func(ctx moleculer.BrokerContext, svc moleculer.Service) {
			if svcHook != nil {
				svcHook(ctx, svc)
			}
			mixin.Started(ctx, svc)
		}
	}
	return service
}

// chainStopped chain the Stope hook of services and mixins
func chainStopped(service moleculer.Service, mixin *moleculer.Mixin) moleculer.Service {
	if mixin.Stopped != nil {
		svcHook := service.Stopped
		service.Stopped = func(ctx moleculer.BrokerContext, svc moleculer.Service) {
			if svcHook != nil {
				svcHook(ctx, svc)
			}
			mixin.Stopped(ctx, svc)
		}
	}
	return service
}

/*
Mixin Strategy:
(done)settings:      	Extend with defaultsDeep.
(done)metadata:   	Extend with defaultsDeep.
(broken)actions:    	Extend with defaultsDeep. You can disable an action from mixin if you set to false in your service.
(done)hooks:      	Extend with defaultsDeep.
(broken)events:     	Concatenate listeners.
TODO:
name:           Merge & overwrite.
version:    	Merge & overwrite.
methods:       	Merge & overwrite.
mixins:	        Merge & overwrite.
dependencies:   Merge & overwrite.
created:    	Concatenate listeners.
started:    	Concatenate listeners.
stopped:    	Concatenate listeners.
*/

func applyMixins(service moleculer.Service) moleculer.Service {
	for _, mixin := range service.Mixins {
		service = extendActions(service, &mixin)
		service = mergeDependencies(service, &mixin)
		service = concatenateEvents(service, &mixin)
		service = extendSettings(service, &mixin)
		service = extendMetadata(service, &mixin)
		service = extendHooks(service, &mixin)
		service = chainCreated(service, &mixin)
		service = chainStarted(service, &mixin)
		service = chainStopped(service, &mixin)
	}
	return service
}

func joinVersionToName(name string, version string) string {
	if version != "" {
		return fmt.Sprintf("%s.%s", version, name)
	}
	return name
}

func CreateServiceEvent(eventName, serviceName, group string, handler moleculer.EventHandler) Event {
	return Event{
		eventName,
		serviceName,
		group,
		handler,
	}
}

func CreateServiceAction(serviceName string, actionName string, handler moleculer.ActionHandler, params moleculer.ActionSchema) Action {
	return Action{
		actionName,
		fmt.Sprintf("%s.%s", serviceName, actionName),
		handler,
		params,
	}
}

// AsMap export the service info in a map containing: name, version, settings, metadata, nodeID, actions and events.
// The events list does not contain internal events (events that starts with $) like $node.disconnected.
func (service *Service) AsMap() map[string]interface{} {
	serviceInfo := make(map[string]interface{})

	serviceInfo["name"] = service.name
	serviceInfo["version"] = service.version

	serviceInfo["settings"] = service.settings
	serviceInfo["metadata"] = service.metadata
	serviceInfo["nodeID"] = service.nodeID

	if service.nodeID == "" {
		panic("no service.nodeID")
	}

	actions := make([]map[string]interface{}, len(service.actions))
	for index, serviceAction := range service.actions {
		actionInfo := make(map[string]interface{})
		actionInfo["name"] = serviceAction.name
		actionInfo["params"] = paramsAsMap(&serviceAction.params)
		actions[index] = actionInfo
	}
	serviceInfo["actions"] = actions

	events := make([]map[string]interface{}, 0)
	for _, serviceEvent := range service.events {
		if !isInternalEvent(serviceEvent) {
			eventInfo := make(map[string]interface{})
			eventInfo["name"] = serviceEvent.name
			eventInfo["serviceName"] = serviceEvent.serviceName
			eventInfo["group"] = serviceEvent.group
			events = append(events, eventInfo)
		}
	}
	serviceInfo["events"] = events
	return serviceInfo
}

func isInternalEvent(event Event) bool {
	return strings.Index(event.Name(), "$") == 0
}

func paramsFromMap(schema interface{}) moleculer.ActionSchema {
	// if schema != nil {
	//mapValues = schema.(map[string]interface{})
	//TODO
	// }
	return moleculer.ObjectSchema{nil}
}

// moleculer.ParamsAsMap converts params schema into a map.
func paramsAsMap(params *moleculer.ActionSchema) map[string]interface{} {
	//TODO
	schema := make(map[string]interface{})
	return schema
}

func (service *Service) AddActionMap(actionInfo map[string]interface{}) *Action {
	action := CreateServiceAction(
		service.fullname,
		actionInfo["name"].(string),
		nil,
		paramsFromMap(actionInfo["schema"]),
	)
	service.actions = append(service.actions, action)
	return &action
}

func (service *Service) RemoveEvent(name string) {
	var newEvents []Event
	for _, event := range service.events {
		if event.name != name {
			newEvents = append(newEvents, event)
		}
	}
	service.events = newEvents
}

func (service *Service) RemoveAction(fullname string) {
	var newActions []Action
	for _, action := range service.actions {
		if action.fullname != fullname {
			newActions = append(newActions, action)
		}
	}
	service.actions = newActions
}

func (service *Service) AddEventMap(eventInfo map[string]interface{}) *Event {
	serviceEvent := Event{
		name:        eventInfo["name"].(string),
		serviceName: eventInfo["serviceName"].(string),
		group:       eventInfo["group"].(string),
	}
	service.events = append(service.events, serviceEvent)
	return &serviceEvent
}

func (service *Service) UpdateFromMap(serviceInfo map[string]interface{}) {
	service.settings = serviceInfo["settings"].(map[string]interface{})
	service.metadata = serviceInfo["metadata"].(map[string]interface{})
}

// populateFromMap populate a service with data from a map[string]interface{}.
func populateFromMap(service *Service, serviceInfo map[string]interface{}) {
	if nodeID, ok := serviceInfo["nodeID"]; ok {
		service.nodeID = nodeID.(string)
	}
	service.version = serviceInfo["version"].(string)
	service.name = serviceInfo["name"].(string)
	service.fullname = joinVersionToName(
		service.name,
		service.version)

	service.settings = serviceInfo["settings"].(map[string]interface{})
	service.metadata = serviceInfo["metadata"].(map[string]interface{})
	actions := serviceInfo["actions"].([]interface{})
	for _, item := range actions {
		actionInfo := item.(map[string]interface{})
		service.AddActionMap(actionInfo)
	}

	events := serviceInfo["events"].([]interface{})
	for _, item := range events {
		eventInfo := item.(map[string]interface{})
		service.AddEventMap(eventInfo)
	}
}

// populateFromSchema populate a service with data from a moleculer.Service.
func (service *Service) populateFromSchema() {
	schema := service.schema
	service.name = schema.Name
	service.version = schema.Version
	service.fullname = joinVersionToName(service.name, service.version)
	service.dependencies = schema.Dependencies
	service.settings = schema.Settings
	if service.settings == nil {
		service.settings = make(map[string]interface{})
	}
	service.metadata = schema.Metadata
	if service.metadata == nil {
		service.metadata = make(map[string]interface{})
	}

	service.actions = make([]Action, len(schema.Actions))
	for index, actionSchema := range schema.Actions {
		service.actions[index] = CreateServiceAction(
			service.fullname,
			actionSchema.Name,
			actionSchema.Handler,
			actionSchema.Schema,
		)
	}

	service.events = make([]Event, len(schema.Events))
	for index, eventSchema := range schema.Events {
		group := eventSchema.Group
		if group == "" {
			group = service.Name()
		}
		service.events[index] = Event{
			name:        eventSchema.Name,
			serviceName: service.Name(),
			group:       group,
			handler:     eventSchema.Handler,
		}
	}

	service.created = schema.Created
	service.started = schema.Started
	service.stopped = schema.Stopped
}

func FromSchema(schema moleculer.Service, logger *log.Entry) *Service {
	if len(schema.Mixins) > 0 {
		schema = applyMixins(schema)
	}
	service := &Service{schema: &schema, logger: logger}
	service.populateFromSchema()
	if service.name == "" {
		panic(errors.New("Service name can't be empty! Maybe it is not a valid Service schema."))
	}

	if service.created != nil {
		go service.created((*service.schema), service.logger)
	}

	return service
}

func CreateServiceFromMap(serviceInfo map[string]interface{}) *Service {
	service := &Service{}
	populateFromMap(service, serviceInfo)
	if service.name == "" {
		panic(errors.New("Service name can't be empty! Maybe it is not a valid Service schema."))
	}
	if service.nodeID == "" {
		panic(errors.New("Service nodeID can't be empty!"))
	}
	return service
}

// Start called by the broker when the service is starting.
func (service *Service) Start(context moleculer.BrokerContext) {
	if service.started != nil {
		go service.started(context, (*service.schema))
	}
}

// Stop called by the broker when the service is stopping.
func (service *Service) Stop(context moleculer.BrokerContext) {
	if service.stopped != nil {
		go service.stopped(context, (*service.schema))
	}
}
