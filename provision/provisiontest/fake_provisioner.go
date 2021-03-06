// Copyright 2015 tsuru authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package provisiontest

import (
	"errors"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/tsuru/config"
	"github.com/tsuru/tsuru/action"
	"github.com/tsuru/tsuru/app/bind"
	"github.com/tsuru/tsuru/provision"
)

var errNotProvisioned = &provision.Error{Reason: "App is not provisioned."}

func init() {
	provision.Register("fake", &FakeProvisioner{})
}

// Fake implementation for provision.App.
type FakeApp struct {
	name           string
	platform       string
	units          []provision.Unit
	logs           []string
	logMut         sync.Mutex
	Commands       []string
	Memory         int64
	Swap           int64
	CpuShare       int
	commMut        sync.Mutex
	ready          bool
	deploys        uint
	env            map[string]bind.EnvVar
	bindCalls      []*provision.Unit
	bindLock       sync.Mutex
	instances      map[string][]bind.ServiceInstance
	UpdatePlatform bool
}

func NewFakeApp(name, platform string, units int) *FakeApp {
	app := FakeApp{
		name:      name,
		platform:  platform,
		units:     make([]provision.Unit, units),
		instances: make(map[string][]bind.ServiceInstance),
	}
	namefmt := "%s-%d"
	for i := 0; i < units; i++ {
		app.units[i] = provision.Unit{
			Name:   fmt.Sprintf(namefmt, name, i),
			Status: provision.StatusStarted,
			Ip:     fmt.Sprintf("10.10.10.%d", i+1),
		}
	}
	return &app
}

func (a *FakeApp) GetMemory() int64 {
	return a.Memory
}

func (a *FakeApp) GetSwap() int64 {
	return a.Swap
}

func (a *FakeApp) GetCpuShare() int {
	return a.CpuShare
}

func (a *FakeApp) HasBind(unit *provision.Unit) bool {
	a.bindLock.Lock()
	defer a.bindLock.Unlock()
	for _, u := range a.bindCalls {
		if u.Name == unit.Name {
			return true
		}
	}
	return false
}

func (a *FakeApp) BindUnit(unit *provision.Unit) error {
	a.bindLock.Lock()
	defer a.bindLock.Unlock()
	a.bindCalls = append(a.bindCalls, unit)
	return nil
}

func (a *FakeApp) UnbindUnit(unit *provision.Unit) error {
	a.bindLock.Lock()
	defer a.bindLock.Unlock()
	index := -1
	for i, u := range a.bindCalls {
		if u.Name == unit.Name {
			index = i
			break
		}
	}
	if index < 0 {
		return errors.New("not bound")
	}
	length := len(a.bindCalls)
	a.bindCalls[index] = a.bindCalls[length-1]
	a.bindCalls = a.bindCalls[:length-1]
	return nil
}

func (a *FakeApp) GetInstances(serviceName string) []bind.ServiceInstance {
	return a.instances[serviceName]
}

func (a *FakeApp) AddInstance(serviceName string, instance bind.ServiceInstance, w io.Writer) error {
	instances := a.instances[serviceName]
	instances = append(instances, instance)
	a.instances[serviceName] = instances
	if w != nil {
		w.Write([]byte("add instance"))
	}
	return nil
}

func (a *FakeApp) RemoveInstance(serviceName string, instance bind.ServiceInstance, w io.Writer) error {
	instances := a.instances[serviceName]
	index := -1
	for i, inst := range instances {
		if inst.Name == instance.Name {
			index = i
			break
		}
	}
	if index < 0 {
		return errors.New("instance not found")
	}
	for i := index; i < len(instances)-1; i++ {
		instances[i] = instances[i+1]
	}
	a.instances[serviceName] = instances[:len(instances)-1]
	if w != nil {
		w.Write([]byte("remove instance"))
	}
	return nil
}

func (a *FakeApp) Logs() []string {
	a.logMut.Lock()
	defer a.logMut.Unlock()
	logs := make([]string, len(a.logs))
	copy(logs, a.logs)
	return logs
}

func (a *FakeApp) HasLog(source, unit, message string) bool {
	log := source + unit + message
	a.logMut.Lock()
	defer a.logMut.Unlock()
	for _, l := range a.logs {
		if l == log {
			return true
		}
	}
	return false
}

func (a *FakeApp) GetCommands() []string {
	a.commMut.Lock()
	defer a.commMut.Unlock()
	return a.Commands
}

func (a *FakeApp) IsReady() bool {
	return a.ready
}

func (a *FakeApp) Ready() error {
	a.ready = true
	return nil
}

func (a *FakeApp) Log(message, source, unit string) error {
	a.logMut.Lock()
	a.logs = append(a.logs, source+unit+message)
	a.logMut.Unlock()
	return nil
}

func (a *FakeApp) GetName() string {
	return a.name
}

func (a *FakeApp) GetPlatform() string {
	return a.platform
}

func (a *FakeApp) GetDeploys() uint {
	return a.deploys
}

func (a *FakeApp) Units() []provision.Unit {
	return a.units
}

func (a *FakeApp) AddUnit(u provision.Unit) {
	a.units = append(a.units, u)
}

func (a *FakeApp) SetEnv(env bind.EnvVar) {
	if a.env == nil {
		a.env = map[string]bind.EnvVar{}
	}
	a.env[env.Name] = env
}

func (a *FakeApp) SetEnvs(envs []bind.EnvVar, publicOnly bool, w io.Writer) error {
	for _, env := range envs {
		a.SetEnv(env)
	}
	return nil
}

func (a *FakeApp) UnsetEnvs(envs []string, publicOnly bool, w io.Writer) error {
	for _, env := range envs {
		delete(a.env, env)
	}
	return nil
}

func (a *FakeApp) GetIp() string {
	return ""
}

func (a *FakeApp) GetUnits() []bind.Unit {
	units := make([]bind.Unit, len(a.units))
	for i, unit := range a.units {
		units[i] = &unit
	}
	return units
}

func (a *FakeApp) InstanceEnv(env string) map[string]bind.EnvVar {
	return nil
}

// Env returns app.Env
func (a *FakeApp) Envs() map[string]bind.EnvVar {
	return a.env
}

func (a *FakeApp) SerializeEnvVars() error {
	a.commMut.Lock()
	a.Commands = append(a.Commands, "serialize")
	a.commMut.Unlock()
	return nil
}

func (a *FakeApp) Restart(w io.Writer) error {
	a.commMut.Lock()
	a.Commands = append(a.Commands, "restart")
	a.commMut.Unlock()
	w.Write([]byte("Restarting app..."))
	return nil
}

func (a *FakeApp) Run(cmd string, w io.Writer, once bool) error {
	a.commMut.Lock()
	a.Commands = append(a.Commands, fmt.Sprintf("ran %s", cmd))
	a.commMut.Unlock()
	return nil
}

func (app *FakeApp) GetUpdatePlatform() bool {
	return app.UpdatePlatform
}

func (app *FakeApp) GetRouter() (string, error) {
	return config.GetString("docker:router")
}

type Cmd struct {
	Cmd  string
	Args []string
	App  provision.App
}

type failure struct {
	method string
	err    error
}

// Fake implementation for provision.Provisioner.
type FakeProvisioner struct {
	cmds     []Cmd
	cmdMut   sync.Mutex
	outputs  chan []byte
	failures chan failure
	apps     map[string]provisionedApp
	mut      sync.RWMutex
	shells   map[string][]provision.ShellOptions
	shellMut sync.Mutex
}

func NewFakeProvisioner() *FakeProvisioner {
	p := FakeProvisioner{}
	p.outputs = make(chan []byte, 8)
	p.failures = make(chan failure, 8)
	p.apps = make(map[string]provisionedApp)
	p.shells = make(map[string][]provision.ShellOptions)
	return &p
}

func (p *FakeProvisioner) getError(method string) error {
	select {
	case fail := <-p.failures:
		if fail.method == method {
			return fail.err
		}
		p.failures <- fail
	default:
	}
	return nil
}

// Restarts returns the number of restarts for a given app.
func (p *FakeProvisioner) Restarts(app provision.App) int {
	p.mut.RLock()
	defer p.mut.RUnlock()
	return p.apps[app.GetName()].restarts
}

// Starts returns the number of starts for a given app.
func (p *FakeProvisioner) Starts(app provision.App) int {
	p.mut.RLock()
	defer p.mut.RUnlock()
	return p.apps[app.GetName()].starts
}

// Stops returns the number of stops for a given app.
func (p *FakeProvisioner) Stops(app provision.App) int {
	p.mut.RLock()
	defer p.mut.RUnlock()
	return p.apps[app.GetName()].stops
}

func (p *FakeProvisioner) CustomData(app provision.App) map[string]interface{} {
	p.mut.RLock()
	defer p.mut.RUnlock()
	return p.apps[app.GetName()].lastData
}

// Shells return all shell calls to the given unit.
func (p *FakeProvisioner) Shells(unit string) []provision.ShellOptions {
	p.shellMut.Lock()
	defer p.shellMut.Unlock()
	return p.shells[unit]
}

// Returns the number of calls to restart.
// GetCmds returns a list of commands executed in an app. If you don't specify
// the command (an empty string), it will return all commands executed in the
// given app.
func (p *FakeProvisioner) GetCmds(cmd string, app provision.App) []Cmd {
	var cmds []Cmd
	p.cmdMut.Lock()
	for _, c := range p.cmds {
		if (cmd == "" || c.Cmd == cmd) && app.GetName() == c.App.GetName() {
			cmds = append(cmds, c)
		}
	}
	p.cmdMut.Unlock()
	return cmds
}

// Provisioned checks whether the given app has been provisioned.
func (p *FakeProvisioner) Provisioned(app provision.App) bool {
	p.mut.RLock()
	defer p.mut.RUnlock()
	_, ok := p.apps[app.GetName()]
	return ok
}

func (p *FakeProvisioner) GetUnits(app provision.App) []provision.Unit {
	p.mut.RLock()
	pApp, _ := p.apps[app.GetName()]
	p.mut.RUnlock()
	return pApp.units
}

// Version returns the last deployed for a given app.
func (p *FakeProvisioner) Version(app provision.App) string {
	p.mut.RLock()
	defer p.mut.RUnlock()
	return p.apps[app.GetName()].version
}

// PrepareOutput sends the given slice of bytes to a queue of outputs.
//
// Each prepared output will be used in the ExecuteCommand. It might be sent to
// the standard output or standard error. See ExecuteCommand docs for more
// details.
func (p *FakeProvisioner) PrepareOutput(b []byte) {
	p.outputs <- b
}

// PrepareFailure prepares a failure for the given method name.
//
// For instance, PrepareFailure("GitDeploy", errors.New("GitDeploy failed")) will
// cause next Deploy call to return the given error. Multiple calls to this
// method will enqueue failures, i.e. three calls to
// PrepareFailure("GitDeploy"...) means that the three next GitDeploy call will
// fail.
func (p *FakeProvisioner) PrepareFailure(method string, err error) {
	p.failures <- failure{method, err}
}

// Reset cleans up the FakeProvisioner, deleting all apps and their data. It
// also deletes prepared failures and output. It's like calling
// NewFakeProvisioner again, without all the allocations.
func (p *FakeProvisioner) Reset() {
	p.cmdMut.Lock()
	p.cmds = nil
	p.cmdMut.Unlock()

	p.mut.Lock()
	p.apps = make(map[string]provisionedApp)
	p.mut.Unlock()

	p.shellMut.Lock()
	p.shells = make(map[string][]provision.ShellOptions)
	p.shellMut.Unlock()

	for {
		select {
		case <-p.outputs:
		case <-p.failures:
		default:
			return
		}
	}
}

func (p *FakeProvisioner) Swap(app1, app2 provision.App) error {
	pApp1, ok := p.apps[app1.GetName()]
	if !ok {
		return errNotProvisioned
	}
	pApp2, ok := p.apps[app2.GetName()]
	if !ok {
		return errNotProvisioned
	}
	pApp1.addr, pApp2.addr = pApp2.addr, pApp1.addr
	p.apps[app1.GetName()] = pApp1
	p.apps[app2.GetName()] = pApp2
	return nil
}

func (p *FakeProvisioner) GitDeploy(app provision.App, version string, w io.Writer) (string, error) {
	if err := p.getError("GitDeploy"); err != nil {
		return "", err
	}
	p.mut.Lock()
	defer p.mut.Unlock()
	pApp, ok := p.apps[app.GetName()]
	if !ok {
		return "", errNotProvisioned
	}
	w.Write([]byte("Git deploy called"))
	pApp.version = version
	p.apps[app.GetName()] = pApp
	return "app-image", nil
}

func (p *FakeProvisioner) ArchiveDeploy(app provision.App, archiveURL string, w io.Writer) (string, error) {
	if err := p.getError("ArchiveDeploy"); err != nil {
		return "", err
	}
	p.mut.Lock()
	defer p.mut.Unlock()
	pApp, ok := p.apps[app.GetName()]
	if !ok {
		return "", errNotProvisioned
	}
	w.Write([]byte("Archive deploy called"))
	pApp.lastArchive = archiveURL
	p.apps[app.GetName()] = pApp
	return "app-image", nil
}

func (p *FakeProvisioner) UploadDeploy(app provision.App, file io.ReadCloser, w io.Writer) (string, error) {
	if err := p.getError("UploadDeploy"); err != nil {
		return "", err
	}
	p.mut.Lock()
	defer p.mut.Unlock()
	pApp, ok := p.apps[app.GetName()]
	if !ok {
		return "", errNotProvisioned
	}
	w.Write([]byte("Upload deploy called"))
	pApp.lastFile = file
	p.apps[app.GetName()] = pApp
	return "app-image", nil
}

func (p *FakeProvisioner) ImageDeploy(app provision.App, img string, w io.Writer) (string, error) {
	if err := p.getError("ImageDeploy"); err != nil {
		return "", err
	}
	p.mut.Lock()
	defer p.mut.Unlock()
	pApp, ok := p.apps[app.GetName()]
	if !ok {
		return "", errNotProvisioned
	}
	w.Write([]byte("Image deploy called"))
	p.apps[app.GetName()] = pApp
	return img, nil
}

func (p *FakeProvisioner) Provision(app provision.App) error {
	if err := p.getError("Provision"); err != nil {
		return err
	}
	if p.Provisioned(app) {
		return &provision.Error{Reason: "App already provisioned."}
	}
	p.mut.Lock()
	defer p.mut.Unlock()
	p.apps[app.GetName()] = provisionedApp{
		app:  app,
		addr: fmt.Sprintf("%s.fake-lb.tsuru.io", app.GetName()),
	}
	return nil
}

func (p *FakeProvisioner) Restart(app provision.App, w io.Writer) error {
	if err := p.getError("Restart"); err != nil {
		return err
	}
	p.mut.Lock()
	defer p.mut.Unlock()
	pApp, ok := p.apps[app.GetName()]
	if !ok {
		return errNotProvisioned
	}
	pApp.restarts++
	p.apps[app.GetName()] = pApp
	if w != nil {
		fmt.Fprintf(w, "restarting app")
	}
	return nil
}

func (p *FakeProvisioner) Start(app provision.App) error {
	p.mut.Lock()
	defer p.mut.Unlock()
	pApp, ok := p.apps[app.GetName()]
	if !ok {
		return errNotProvisioned
	}
	pApp.starts++
	p.apps[app.GetName()] = pApp
	return nil
}

func (p *FakeProvisioner) Destroy(app provision.App) error {
	if err := p.getError("Destroy"); err != nil {
		return err
	}
	if !p.Provisioned(app) {
		return errNotProvisioned
	}
	p.mut.Lock()
	defer p.mut.Unlock()
	delete(p.apps, app.GetName())
	return nil
}

func (p *FakeProvisioner) AddUnits(app provision.App, n uint, w io.Writer) ([]provision.Unit, error) {
	if err := p.getError("AddUnits"); err != nil {
		return nil, err
	}
	if n == 0 {
		return nil, errors.New("Cannot add 0 units.")
	}
	p.mut.Lock()
	defer p.mut.Unlock()
	pApp, ok := p.apps[app.GetName()]
	if !ok {
		return nil, errNotProvisioned
	}
	name := app.GetName()
	platform := app.GetPlatform()
	length := uint(len(pApp.units))
	for i := uint(0); i < n; i++ {
		unit := provision.Unit{
			Name:    fmt.Sprintf("%s-%d", name, pApp.unitLen),
			AppName: name,
			Type:    platform,
			Status:  provision.StatusStarted,
			Ip:      fmt.Sprintf("10.10.10.%d", length+i+1),
		}
		pApp.units = append(pApp.units, unit)
		pApp.unitLen++
	}
	result := make([]provision.Unit, int(n))
	copy(result, pApp.units[length:])
	p.apps[app.GetName()] = pApp
	if w != nil {
		fmt.Fprintf(w, "added %d units", n)
	}
	return result, nil
}

func (p *FakeProvisioner) RemoveUnits(app provision.App, n uint) error {
	if err := p.getError("RemoveUnits"); err != nil {
		return err
	}
	p.mut.Lock()
	defer p.mut.Unlock()
	pApp, ok := p.apps[app.GetName()]
	if !ok {
		return errNotProvisioned
	}
	if n >= uint(len(pApp.units)) {
		return errors.New("too many units to remove")
	}
	pApp.units = pApp.units[int(n):]
	pApp.unitLen -= int(n)
	p.apps[app.GetName()] = pApp
	return nil
}

func (p *FakeProvisioner) RemoveUnit(unit provision.Unit) error {
	if err := p.getError("RemoveUnit"); err != nil {
		return err
	}
	p.mut.Lock()
	defer p.mut.Unlock()
	app, ok := p.apps[unit.AppName]
	if !ok {
		return errNotProvisioned
	}
	index := -1
	for i, u := range app.units {
		if u.Name == unit.Name {
			index = i
		}
	}
	if index < 0 {
		return errors.New("unit not found")
	}
	app.units[index] = app.units[len(app.units)-1]
	app.units = app.units[:len(app.units)-1]
	app.unitLen--
	p.apps[unit.AppName] = app
	return nil
}

// ExecuteCommand will pretend to execute the given command, recording data
// about it.
//
// The output of the command must be prepared with PrepareOutput, and failures
// must be prepared with PrepareFailure. In case of failure, the prepared
// output will be sent to the standard error stream, otherwise, it will be sent
// to the standard error stream.
//
// When there is no output nor failure prepared, ExecuteCommand will return a
// timeout error.
func (p *FakeProvisioner) ExecuteCommand(stdout, stderr io.Writer, app provision.App, cmd string, args ...string) error {
	var (
		output []byte
		err    error
	)
	command := Cmd{
		Cmd:  cmd,
		Args: args,
		App:  app,
	}
	p.cmdMut.Lock()
	p.cmds = append(p.cmds, command)
	p.cmdMut.Unlock()
	for range app.Units() {
		select {
		case output = <-p.outputs:
			select {
			case fail := <-p.failures:
				if fail.method == "ExecuteCommand" {
					stderr.Write(output)
					return fail.err
				}
				p.failures <- fail
			default:
				stdout.Write(output)
			}
		case fail := <-p.failures:
			if fail.method == "ExecuteCommand" {
				err = fail.err
				select {
				case output = <-p.outputs:
					stderr.Write(output)
				default:
				}
			} else {
				p.failures <- fail
			}
		case <-time.After(2e9):
			return errors.New("FakeProvisioner timed out waiting for output.")
		}
	}
	return err
}

func (p *FakeProvisioner) ExecuteCommandOnce(stdout, stderr io.Writer, app provision.App, cmd string, args ...string) error {
	var output []byte
	command := Cmd{
		Cmd:  cmd,
		Args: args,
		App:  app,
	}
	p.cmdMut.Lock()
	p.cmds = append(p.cmds, command)
	p.cmdMut.Unlock()
	select {
	case output = <-p.outputs:
		stdout.Write(output)
	case fail := <-p.failures:
		if fail.method == "ExecuteCommandOnce" {
			select {
			case output = <-p.outputs:
				stderr.Write(output)
			default:
			}
			return fail.err
		} else {
			p.failures <- fail
		}
	case <-time.After(2e9):
		return errors.New("FakeProvisioner timed out waiting for output.")
	}
	return nil
}

func (p *FakeProvisioner) AddUnit(app provision.App, unit provision.Unit) {
	p.mut.Lock()
	defer p.mut.Unlock()
	a := p.apps[app.GetName()]
	a.units = append(a.units, unit)
	a.unitLen++
	p.apps[app.GetName()] = a
}

func (p *FakeProvisioner) Units(app provision.App) []provision.Unit {
	p.mut.Lock()
	defer p.mut.Unlock()
	return p.apps[app.GetName()].units
}

func (p *FakeProvisioner) SetUnitStatus(unit provision.Unit, status provision.Status) error {
	p.mut.Lock()
	defer p.mut.Unlock()
	app, ok := p.apps[unit.AppName]
	if !ok {
		return errNotProvisioned
	}
	index := -1
	for i, unt := range app.units {
		if unt.Name == unit.Name {
			index = i
			break
		}
	}
	if index < 0 {
		return errors.New("unit not found")
	}
	app.units[index].Status = status
	p.apps[unit.AppName] = app
	return nil
}

func (p *FakeProvisioner) Addr(app provision.App) (string, error) {
	if err := p.getError("Addr"); err != nil {
		return "", err
	}
	pApp, ok := p.apps[app.GetName()]
	if !ok {
		return "", errNotProvisioned
	}
	return pApp.addr, nil
}

func (p *FakeProvisioner) SetCName(app provision.App, cname string) error {
	if err := p.getError("SetCName"); err != nil {
		return err
	}
	p.mut.Lock()
	defer p.mut.Unlock()
	pApp, ok := p.apps[app.GetName()]
	if !ok {
		return errNotProvisioned
	}
	pApp.cnames = append(pApp.cnames, cname)
	p.apps[app.GetName()] = pApp
	return nil
}

func (p *FakeProvisioner) UnsetCName(app provision.App, cname string) error {
	if err := p.getError("UnsetCName"); err != nil {
		return err
	}
	p.mut.Lock()
	defer p.mut.Unlock()
	pApp, ok := p.apps[app.GetName()]
	if !ok {
		return errNotProvisioned
	}
	pApp.cnames = []string{}
	p.apps[app.GetName()] = pApp
	return nil
}

func (p *FakeProvisioner) HasCName(app provision.App, cname string) bool {
	p.mut.RLock()
	pApp, ok := p.apps[app.GetName()]
	p.mut.RUnlock()
	for _, cnameApp := range pApp.cnames {
		if cnameApp == cname {
			return ok && true
		}
	}
	return false
}

func (p *FakeProvisioner) Stop(app provision.App) error {
	p.mut.Lock()
	defer p.mut.Unlock()
	pApp, ok := p.apps[app.GetName()]
	if !ok {
		return errNotProvisioned
	}
	pApp.stops++
	for i, u := range pApp.units {
		u.Status = provision.StatusStopped
		pApp.units[i] = u
	}
	p.apps[app.GetName()] = pApp
	return nil
}

func (p *FakeProvisioner) RegisterUnit(unit provision.Unit, customData map[string]interface{}) error {
	p.mut.Lock()
	defer p.mut.Unlock()
	a, ok := p.apps[unit.AppName]
	if !ok {
		return errors.New("app not found")
	}
	a.lastData = customData
	for i, u := range a.units {
		if u.Name == unit.Name {
			u.Ip = u.Ip + "-updated"
			a.units[i] = u
			p.apps[unit.AppName] = a
			return nil
		}
	}
	return errors.New("unit not found")
}

func (p *FakeProvisioner) Shell(opts provision.ShellOptions) error {
	var unit provision.Unit
	units := p.Units(opts.App)
	if len(units) == 0 {
		return errors.New("app has no units")
	} else if opts.Unit != "" {
		for _, u := range units {
			if u.Name == opts.Unit {
				unit = u
				break
			}
		}
	} else {
		unit = units[0]
	}
	if unit.Name == "" {
		return errors.New("unit not found")
	}
	p.shellMut.Lock()
	defer p.shellMut.Unlock()
	p.shells[unit.Name] = append(p.shells[unit.Name], opts)
	return nil
}

func (p *FakeProvisioner) ValidAppImages(appName string) ([]string, error) {
	if err := p.getError("ValidAppImages"); err != nil {
		return nil, err
	}
	return []string{"app-image-old", "app-image"}, nil
}

type PipelineFakeProvisioner struct {
	*FakeProvisioner
	executedPipeline bool
}

func (p *PipelineFakeProvisioner) ExecutedPipeline() bool {
	return p.executedPipeline
}

func (p *PipelineFakeProvisioner) DeployPipeline() *action.Pipeline {
	act := action.Action{
		Name: "change-executed-pipeline",
		Forward: func(ctx action.FWContext) (action.Result, error) {
			p.executedPipeline = true
			return nil, nil
		},
		Backward: func(ctx action.BWContext) {
		},
	}
	actions := []*action.Action{&act}
	pipeline := action.NewPipeline(actions...)
	return pipeline
}

type PipelineErrorFakeProvisioner struct {
	*FakeProvisioner
}

func (p *PipelineErrorFakeProvisioner) DeployPipeline() *action.Pipeline {
	act := action.Action{
		Name: "error-pipeline",
		Forward: func(ctx action.FWContext) (action.Result, error) {
			return nil, errors.New("deploy error")
		},
		Backward: func(ctx action.BWContext) {
		},
	}
	actions := []*action.Action{&act}
	pipeline := action.NewPipeline(actions...)
	return pipeline
}

type ExtensibleFakeProvisioner struct {
	*FakeProvisioner
	platforms []provisionedPlatform
}

func (p *ExtensibleFakeProvisioner) GetPlatform(name string) *provisionedPlatform {
	_, platform := p.getPlatform(name)
	return platform
}

func (p *ExtensibleFakeProvisioner) getPlatform(name string) (int, *provisionedPlatform) {
	for i, platform := range p.platforms {
		if platform.Name == name {
			return i, &platform
		}
	}
	return -1, nil
}

func (p *ExtensibleFakeProvisioner) PlatformAdd(name string, args map[string]string, w io.Writer) error {
	if err := p.getError("PlatformAdd"); err != nil {
		return err
	}
	if p.GetPlatform(name) != nil {
		return errors.New("duplicate platform")
	}
	p.platforms = append(p.platforms, provisionedPlatform{Name: name, Args: args, Version: 1})
	return nil
}

func (p *ExtensibleFakeProvisioner) PlatformUpdate(name string, args map[string]string, w io.Writer) error {
	index, platform := p.getPlatform(name)
	if platform == nil {
		return errors.New("platform not found")
	}
	platform.Version += 1
	platform.Args = args
	p.platforms[index] = *platform
	return nil
}

func (p *ExtensibleFakeProvisioner) PlatformRemove(name string) error {
	index, _ := p.getPlatform(name)
	if index < 0 {
		return errors.New("platform not found")
	}
	p.platforms[index] = p.platforms[len(p.platforms)-1]
	p.platforms = p.platforms[:len(p.platforms)-1]
	return nil
}

type provisionedApp struct {
	units       []provision.Unit
	app         provision.App
	restarts    int
	starts      int
	stops       int
	version     string
	lastArchive string
	lastFile    io.ReadCloser
	cnames      []string
	addr        string
	unitLen     int
	lastData    map[string]interface{}
}

type provisionedPlatform struct {
	Name    string
	Args    map[string]string
	Version int
}
