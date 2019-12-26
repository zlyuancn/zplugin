/*
-------------------------------------------------
   Author :       Zhang Fan
   date：         2019/12/26
   Description :
-------------------------------------------------
*/

package zplugin

import (
    "container/list"
    "fmt"
    "sync"
)

// 插件接口
type Pluginer interface {
    On() error
    Off()
}

// 插件条目
type plugin_entry struct {
    name   string
    plugin Pluginer
    status Status
}

type Status int

const (
    StatusOff Status = iota
    StatusOn
)

type PluginManage struct {
    mx          sync.Mutex
    plugins     *list.List               // 注册顺序
    plugin_mm   map[string]*list.Element // 注册的插件
    pluginons   *list.List               // 启动顺序
    pluginon_mm map[string]*list.Element // 启动的插件
}

func New() *PluginManage {
    return &PluginManage{
        plugins:     list.New(),
        plugin_mm:   make(map[string]*list.Element),
        pluginons:   list.New(),
        pluginon_mm: make(map[string]*list.Element),
    }
}

// 注册插件
// on表示是否立即启动
func (m *PluginManage) RegistryPlugin(name string, plugin Pluginer, on bool) error {
    m.mx.Lock()
    defer m.mx.Unlock()

    if _, ok := m.plugin_mm[name]; ok {
        return fmt.Errorf("插件名<%s>已被注册", name)
    }

    pe := &plugin_entry{
        name:   name,
        plugin: plugin,
        status: StatusOff,
    }

    el := m.plugins.PushBack(pe)
    m.plugin_mm[name] = el

    if on {
        if err := m.on(pe); err != nil {
            return fmt.Errorf("插件<%s>启动失败: %s", name, err)
        }
    }

    return nil
}

// 取消注册插件, 如果该插件已开启在取消注册时会关闭它
func (m *PluginManage) UnRegistryPlugin(name string) error {
    m.mx.Lock()
    defer m.mx.Unlock()

    el, ok := m.plugin_mm[name]
    if !ok {
        return fmt.Errorf("插件<%s>不存在", name)
    }

    pe := el.Value.(*plugin_entry)
    m.plugins.Remove(el)
    delete(m.plugin_mm, pe.name)
    if pe.status == StatusOn {
        m.off(pe)
    }
    return nil
}

// 按注册顺序启动所有插件, 如果某个插件启动失败会停止启动并立即返回错误
func (m *PluginManage) On() error {
    m.mx.Lock()
    defer m.mx.Unlock()

    el := m.plugins.Front()
    for {
        if el == nil {
            return nil
        }

        pe := el.Value.(*plugin_entry)
        if pe.status == StatusOff {
            if err := m.on(pe); err != nil {
                return fmt.Errorf("插件<%s>启动失败: %s", pe.name, err)
            }
        }

        el = el.Next()
    }
}

// 按启动顺序倒序关闭所有插件
func (m *PluginManage) Off() {
    m.mx.Lock()
    defer m.mx.Unlock()

    el := m.pluginons.Back()
    for {
        if el == nil {
            return
        }

        pe := el.Value.(*plugin_entry)
        if pe.status == StatusOn {
            m.off(pe)
        }
        el = el.Prev()
    }
}

// 获取插件
func (m *PluginManage) Get(name string) (Pluginer, error) {
    m.mx.Lock()
    defer m.mx.Unlock()

    el, ok := m.plugin_mm[name]
    if !ok {
        return nil, fmt.Errorf("插件<%s>不存在", name)
    }

    return el.Value.(*plugin_entry).plugin, nil
}

// 获取某个插件是否已开启
func (m *PluginManage) IsOn(name string) (bool, error) {
    m.mx.Lock()
    defer m.mx.Unlock()

    el, ok := m.plugin_mm[name]
    if !ok {
        return false, fmt.Errorf("插件<%s>不存在", name)
    }
    return el.Value.(*plugin_entry).status == StatusOn, nil
}

func (m *PluginManage) on(pe *plugin_entry) error {
    if err := pe.plugin.On(); err != nil {
        return err
    }
    pe.status = StatusOn

    el := m.pluginons.PushBack(pe)
    m.pluginon_mm[pe.name] = el
    return nil
}

func (m *PluginManage) off(pe *plugin_entry) {
    el := m.pluginon_mm[pe.name]
    m.pluginons.Remove(el)
    delete(m.pluginon_mm, pe.name)

    pe.status = StatusOff
    pe.plugin.Off()
}
