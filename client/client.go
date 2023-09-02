/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements.  See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License.  You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package client

import (
	"context"
	commonCfg "dubbo.apache.org/dubbo-go/v3/common/config"
	"dubbo.apache.org/dubbo-go/v3/config"
	"dubbo.apache.org/dubbo-go/v3/global"
)

import (
	"github.com/pkg/errors"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/protocol"
	invocation_impl "dubbo.apache.org/dubbo-go/v3/protocol/invocation"
)

type Client struct {
	invoker protocol.Invoker
	info    *ClientInfo
	cfg     *ReferenceConfig

	cliOpts *ClientOptions
}

type ClientInfo struct {
	InterfaceName    string
	MethodNames      []string
	ClientInjectFunc func(dubboCliRaw interface{}, cli *Client)
	Meta             map[string]interface{}
}

func (cli *Client) init(opts ...global.ReferenceOption) error {
	refCfg := global.DefaultReferenceConfig()
	for _, opt := range opts {
		opt(refCfg)
	}

	// init method
	if length := len(refCfg.Methods); length > 0 {
		cli.methodsCompat = make([]*config.MethodConfig, length)
		for i, method := range refCfg.Methods {
			cli.methodsCompat[i] = compatMethodConfig(method)
			if err := cli.methodsCompat[i].Init(); err != nil {
				return err
			}
		}

	}
	// init application
	if refCfg.pplication != nil {
		rc.applicationCompat = compatApplicationConfig(rc.application)
		if err := rc.applicationCompat.Init(); err != nil {
			return err
		}
		rc.metaDataType = rc.applicationCompat.MetadataType
		if rc.Group == "" {
			rc.Group = rc.applicationCompat.Group
		}
		if rc.Version == "" {
			rc.Version = rc.applicationCompat.Version
		}
	}
	// init cluster
	if rc.Cluster == "" {
		rc.Cluster = "failover"
	}
	// todo: move to registry package
	// init registries
	if rc.registries != nil {
		rc.registriesCompat = make(map[string]*config.RegistryConfig)
		for key, reg := range rc.registries {
			rc.registriesCompat[key] = compatRegistryConfig(reg)
			if err := rc.registriesCompat[key].Init(); err != nil {
				return err
			}
		}
	}
	rc.RegistryIDs = commonCfg.TranslateIds(rc.RegistryIDs)

	return commonCfg.Verify(rc)
}

func (cli *Client) call(ctx context.Context, paramsRawVals []interface{}, interfaceName, methodName, callType string, opts ...CallOption) (protocol.Result, error) {
	// get a default CallOptions
	// apply CallOption
	options := newDefaultCallOptions()
	for _, opt := range opts {
		opt(options)
	}

	inv, err := generateInvocation(methodName, paramsRawVals, callType, options)
	if err != nil {
		return nil, err
	}
	// todo: move timeout into context or invocation
	return cli.invoker.Invoke(ctx, inv), nil

}

func (cli *Client) CallUnary(ctx context.Context, req, resp interface{}, interfaceName, methodName string, opts ...CallOption) error {
	res, err := cli.call(ctx, []interface{}{req, resp}, interfaceName, methodName, constant.CallUnary, opts...)
	if err != nil {
		return err
	}
	return res.Error()
}

func (cli *Client) CallClientStream(ctx context.Context, interfaceName, methodName string, opts ...CallOption) (interface{}, error) {
	res, err := cli.call(ctx, nil, interfaceName, methodName, constant.CallClientStream, opts...)
	if err != nil {
		return nil, err
	}
	return res.Result(), res.Error()
}

func (cli *Client) CallServerStream(ctx context.Context, req interface{}, interfaceName, methodName string, opts ...CallOption) (interface{}, error) {
	res, err := cli.call(ctx, []interface{}{req}, interfaceName, methodName, constant.CallServerStream, opts...)
	if err != nil {
		return nil, err
	}
	return res.Result(), res.Error()
}

func (cli *Client) CallBidiStream(ctx context.Context, interfaceName, methodName string, opts ...CallOption) (interface{}, error) {
	res, err := cli.call(ctx, nil, interfaceName, methodName, constant.CallBidiStream, opts...)
	if err != nil {
		return nil, err
	}
	return res.Result(), res.Error()
}

func (cli *Client) Init(info *ClientInfo) error {
	if info == nil {
		return errors.New("ClientInfo is nil")
	}
	cli.cfg.ReferWithInfo(info)
	cli.invoker = cli.cfg.invoker

	return nil
}

func generateInvocation(methodName string, paramsRawVals []interface{}, callType string, opts *CallOptions) (protocol.Invocation, error) {
	inv := invocation_impl.NewRPCInvocationWithOptions(
		invocation_impl.WithMethodName(methodName),
		// todo: process opts
		invocation_impl.WithParameterRawValues(paramsRawVals),
	)
	inv.SetAttribute(constant.CallTypeKey, callType)

	return inv, nil
}

func NewClient(opts ...ReferenceOption) (*Client, error) {
	// todo(DMwangnima): create a default ReferenceConfig
	newRefCfg := &ReferenceConfig{}
	if err := newRefCfg.Init(opts...); err != nil {
		return nil, err
	}
	return &Client{
		cfg: newRefCfg,
	}, nil
}
