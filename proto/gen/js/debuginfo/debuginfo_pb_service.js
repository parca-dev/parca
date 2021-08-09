// package: parca.debuginfo
// file: debuginfo/debuginfo.proto

var debuginfo_debuginfo_pb = require("../debuginfo/debuginfo_pb");
var grpc = require("@improbable-eng/grpc-web").grpc;

var DebugInfo = (function () {
  function DebugInfo() {}
  DebugInfo.serviceName = "parca.debuginfo.DebugInfo";
  return DebugInfo;
}());

DebugInfo.Exists = {
  methodName: "Exists",
  service: DebugInfo,
  requestStream: false,
  responseStream: false,
  requestType: debuginfo_debuginfo_pb.DebugInfoExistsRequest,
  responseType: debuginfo_debuginfo_pb.DebugInfoExistsResponse
};

DebugInfo.Upload = {
  methodName: "Upload",
  service: DebugInfo,
  requestStream: true,
  responseStream: false,
  requestType: debuginfo_debuginfo_pb.DebugInfoUploadRequest,
  responseType: debuginfo_debuginfo_pb.DebugInfoUploadResponse
};

exports.DebugInfo = DebugInfo;

function DebugInfoClient(serviceHost, options) {
  this.serviceHost = serviceHost;
  this.options = options || {};
}

DebugInfoClient.prototype.exists = function exists(requestMessage, metadata, callback) {
  if (arguments.length === 2) {
    callback = arguments[1];
  }
  var client = grpc.unary(DebugInfo.Exists, {
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
    }
  });
  return {
    cancel: function () {
      callback = null;
      client.close();
    }
  };
};

DebugInfoClient.prototype.upload = function upload(metadata) {
  var listeners = {
    end: [],
    status: []
  };
  var client = grpc.client(DebugInfo.Upload, {
    host: this.serviceHost,
    metadata: metadata,
    transport: this.options.transport
  });
  client.onEnd(function (status, statusMessage, trailers) {
    listeners.status.forEach(function (handler) {
      handler({ code: status, details: statusMessage, metadata: trailers });
    });
    listeners.end.forEach(function (handler) {
      handler({ code: status, details: statusMessage, metadata: trailers });
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
    }
  };
};

exports.DebugInfoClient = DebugInfoClient;

