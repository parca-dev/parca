// package: parca.debuginfo.v1alpha1
// file: parca/debuginfo/v1alpha1/debuginfo.proto

var parca_debuginfo_v1alpha1_debuginfo_pb = require('../../../parca/debuginfo/v1alpha1/debuginfo_pb');
var grpc = require('@improbable-eng/grpc-web').grpc;

var DebugInfoService = (function () {
  function DebugInfoService() {}
  DebugInfoService.serviceName = 'parca.debuginfo.v1alpha1.DebugInfoService';
  return DebugInfoService;
})();

DebugInfoService.Exists = {
  methodName: 'Exists',
  service: DebugInfoService,
  requestStream: false,
  responseStream: false,
  requestType: parca_debuginfo_v1alpha1_debuginfo_pb.ExistsRequest,
  responseType: parca_debuginfo_v1alpha1_debuginfo_pb.ExistsResponse,
};

DebugInfoService.Upload = {
  methodName: 'Upload',
  service: DebugInfoService,
  requestStream: true,
  responseStream: false,
  requestType: parca_debuginfo_v1alpha1_debuginfo_pb.UploadRequest,
  responseType: parca_debuginfo_v1alpha1_debuginfo_pb.UploadResponse,
};

exports.DebugInfoService = DebugInfoService;

function DebugInfoServiceClient(serviceHost, options) {
  this.serviceHost = serviceHost;
  this.options = options || {};
}

DebugInfoServiceClient.prototype.exists = function exists(requestMessage, metadata, callback) {
  if (arguments.length === 2) {
    callback = arguments[1];
  }
  var client = grpc.unary(DebugInfoService.Exists, {
    request: requestMessage,
    host: this.serviceHost,
    metadata: metadata,
    transport: this.options.transport,
    debug: this.options.debug,
    onEnd: function (response) {
      if (callback) {
        if (response.status !== grpc.Code.OK) {
          var err = new Error(response.statusMessage);
          err.code = response.status;
          err.metadata = response.trailers;
          callback(err, null);
        } else {
          callback(null, response.message);
        }
      }
    },
  });
  return {
    cancel: function () {
      callback = null;
      client.close();
    },
  };
};

DebugInfoServiceClient.prototype.upload = function upload(metadata) {
  var listeners = {
    end: [],
    status: [],
  };
  var client = grpc.client(DebugInfoService.Upload, {
    host: this.serviceHost,
    metadata: metadata,
    transport: this.options.transport,
  });
  client.onEnd(function (status, statusMessage, trailers) {
    listeners.status.forEach(function (handler) {
      handler({code: status, details: statusMessage, metadata: trailers});
    });
    listeners.end.forEach(function (handler) {
      handler({code: status, details: statusMessage, metadata: trailers});
    });
    listeners = null;
  });
  return {
    on: function (type, handler) {
      listeners[type].push(handler);
      return this;
    },
    write: function (requestMessage) {
      if (!client.started) {
        client.start(metadata);
      }
      client.send(requestMessage);
      return this;
    },
    end: function () {
      client.finishSend();
    },
    cancel: function () {
      listeners = null;
      client.close();
    },
  };
};

exports.DebugInfoServiceClient = DebugInfoServiceClient;
