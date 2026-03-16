package nacos

import (
	"github.com/nacos-group/nacos-sdk-go/model"
	"github.com/nacos-group/nacos-sdk-go/vo"
)

type mockNamingClient struct {
	registerInstanceFn     func(vo.RegisterInstanceParam) (bool, error)
	deregisterInstanceFn   func(vo.DeregisterInstanceParam) (bool, error)
	updateInstanceFn       func(vo.UpdateInstanceParam) (bool, error)
	getServiceFn           func(vo.GetServiceParam) (model.Service, error)
	selectAllInstancesFn   func(vo.SelectAllInstancesParam) ([]model.Instance, error)
	selectInstancesFn      func(vo.SelectInstancesParam) ([]model.Instance, error)
	selectOneHealthyInstFn func(vo.SelectOneHealthInstanceParam) (*model.Instance, error)
	subscribeFn            func(*vo.SubscribeParam) error
	unsubscribeFn          func(*vo.SubscribeParam) error
	getAllServicesInfoFn   func(vo.GetAllServiceInfoParam) (model.ServiceList, error)
}

func (m *mockNamingClient) RegisterInstance(param vo.RegisterInstanceParam) (bool, error) {
	if m.registerInstanceFn != nil {
		return m.registerInstanceFn(param)
	}
	return true, nil
}

func (m *mockNamingClient) DeregisterInstance(param vo.DeregisterInstanceParam) (bool, error) {
	if m.deregisterInstanceFn != nil {
		return m.deregisterInstanceFn(param)
	}
	return true, nil
}

func (m *mockNamingClient) UpdateInstance(param vo.UpdateInstanceParam) (bool, error) {
	if m.updateInstanceFn != nil {
		return m.updateInstanceFn(param)
	}
	return true, nil
}

func (m *mockNamingClient) GetService(param vo.GetServiceParam) (model.Service, error) {
	if m.getServiceFn != nil {
		return m.getServiceFn(param)
	}
	return model.Service{}, nil
}

func (m *mockNamingClient) SelectAllInstances(param vo.SelectAllInstancesParam) ([]model.Instance, error) {
	if m.selectAllInstancesFn != nil {
		return m.selectAllInstancesFn(param)
	}
	return nil, nil
}

func (m *mockNamingClient) SelectInstances(param vo.SelectInstancesParam) ([]model.Instance, error) {
	if m.selectInstancesFn != nil {
		return m.selectInstancesFn(param)
	}
	return nil, nil
}

func (m *mockNamingClient) SelectOneHealthyInstance(param vo.SelectOneHealthInstanceParam) (*model.Instance, error) {
	if m.selectOneHealthyInstFn != nil {
		return m.selectOneHealthyInstFn(param)
	}
	return nil, nil
}

func (m *mockNamingClient) Subscribe(param *vo.SubscribeParam) error {
	if m.subscribeFn != nil {
		return m.subscribeFn(param)
	}
	return nil
}

func (m *mockNamingClient) Unsubscribe(param *vo.SubscribeParam) error {
	if m.unsubscribeFn != nil {
		return m.unsubscribeFn(param)
	}
	return nil
}

func (m *mockNamingClient) GetAllServicesInfo(param vo.GetAllServiceInfoParam) (model.ServiceList, error) {
	if m.getAllServicesInfoFn != nil {
		return m.getAllServicesInfoFn(param)
	}
	return model.ServiceList{}, nil
}
