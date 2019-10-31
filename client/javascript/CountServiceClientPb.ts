/**
 * @fileoverview gRPC-Web generated client stub for count
 * @enhanceable
 * @public
 */

// GENERATED CODE -- DO NOT EDIT!


import * as grpcWeb from 'grpc-web';

import {
  CountRequest,
  CountResponse} from './count_pb';

export class CountClient {
  client_: grpcWeb.AbstractClientBase;
  hostname_: string;
  credentials_: null | { [index: string]: string; };
  options_: null | { [index: string]: string; };

  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: string; }) {
    if (!options) options = {};
    if (!credentials) credentials = {};
    options['format'] = 'text';

    this.client_ = new grpcWeb.GrpcWebClientBase(options);
    this.hostname_ = hostname;
    this.credentials_ = credentials;
    this.options_ = options;
  }

  methodInfoCount = new grpcWeb.AbstractClientBase.MethodInfo(
    CountResponse,
    (request: CountRequest) => {
      return request.serializeBinary();
    },
    CountResponse.deserializeBinary
  );

  count(
    request: CountRequest,
    metadata: grpcWeb.Metadata | null,
    callback: (err: grpcWeb.Error,
               response: CountResponse) => void) {
    return this.client_.rpcCall(
      this.hostname_ +
        '/count.Count/Count',
      request,
      metadata || {},
      this.methodInfoCount,
      callback);
  }

}

