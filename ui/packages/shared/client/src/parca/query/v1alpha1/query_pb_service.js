// package: parca.query.v1alpha1
// file: parca/query/v1alpha1/query.proto

var parca_query_v1alpha1_query_pb = require("../../../parca/query/v1alpha1/query_pb");
var grpc = require("@improbable-eng/grpc-web").grpc;

var QueryService = (function () {
  function QueryService() {}
  QueryService.serviceName = "parca.query.v1alpha1.QueryService";
  return QueryService;
}());

QueryService.QueryRange = {
  methodName: "QueryRange",
  service: QueryService,
  requestStream: false,
  responseStream: false,
  requestType: parca_query_v1alpha1_query_pb.QueryRangeRequest,
  responseType: parca_query_v1alpha1_query_pb.QueryRangeResponse
};

QueryService.Query = {
  methodName: "Query",
  service: QueryService,
  requestStream: false,
  responseStream: false,
  requestType: parca_query_v1alpha1_query_pb.QueryRequest,
  responseType: parca_query_v1alpha1_query_pb.QueryResponse
};

QueryService.Series = {
  methodName: "Series",
  service: QueryService,
  requestStream: false,
  responseStream: false,
  requestType: parca_query_v1alpha1_query_pb.SeriesRequest,
  responseType: parca_query_v1alpha1_query_pb.SeriesResponse
};

QueryService.Labels = {
  methodName: "Labels",
  service: QueryService,
  requestStream: false,
  responseStream: false,
  requestType: parca_query_v1alpha1_query_pb.LabelsRequest,
  responseType: parca_query_v1alpha1_query_pb.LabelsResponse
};

QueryService.Values = {
  methodName: "Values",
  service: QueryService,
  requestStream: false,
  responseStream: false,
  requestType: parca_query_v1alpha1_query_pb.ValuesRequest,
  responseType: parca_query_v1alpha1_query_pb.ValuesResponse
};

exports.QueryService = QueryService;

function QueryServiceClient(serviceHost, options) {
  this.serviceHost = serviceHost;
  this.options = options || {};
}

QueryServiceClient.prototype.queryRange = function queryRange(requestMessage, metadata, callback) {
  if (arguments.length === 2) {
    callback = arguments[1];
  }
  var client = grpc.unary(QueryService.QueryRange, {
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

QueryServiceClient.prototype.query = function query(requestMessage, metadata, callback) {
  if (arguments.length === 2) {
    callback = arguments[1];
  }
  var client = grpc.unary(QueryService.Query, {
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

QueryServiceClient.prototype.series = function series(requestMessage, metadata, callback) {
  if (arguments.length === 2) {
    callback = arguments[1];
  }
  var client = grpc.unary(QueryService.Series, {
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

QueryServiceClient.prototype.labels = function labels(requestMessage, metadata, callback) {
  if (arguments.length === 2) {
    callback = arguments[1];
  }
  var client = grpc.unary(QueryService.Labels, {
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

QueryServiceClient.prototype.values = function values(requestMessage, metadata, callback) {
  if (arguments.length === 2) {
    callback = arguments[1];
  }
  var client = grpc.unary(QueryService.Values, {
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

exports.QueryServiceClient = QueryServiceClient;

