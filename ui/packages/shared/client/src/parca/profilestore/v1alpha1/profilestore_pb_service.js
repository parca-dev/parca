// package: parca.profilestore.v1alpha1
// file: parca/profilestore/v1alpha1/profilestore.proto

var parca_profilestore_v1alpha1_profilestore_pb = require("../../../parca/profilestore/v1alpha1/profilestore_pb");
var grpc = require("@improbable-eng/grpc-web").grpc;

var ProfileStoreService = (function () {
  function ProfileStoreService() {}
  ProfileStoreService.serviceName = "parca.profilestore.v1alpha1.ProfileStoreService";
  return ProfileStoreService;
}());

ProfileStoreService.WriteRaw = {
  methodName: "WriteRaw",
  service: ProfileStoreService,
  requestStream: false,
  responseStream: false,
  requestType: parca_profilestore_v1alpha1_profilestore_pb.WriteRawRequest,
  responseType: parca_profilestore_v1alpha1_profilestore_pb.WriteRawResponse
};

exports.ProfileStoreService = ProfileStoreService;

function ProfileStoreServiceClient(serviceHost, options) {
  this.serviceHost = serviceHost;
  this.options = options || {};
}

ProfileStoreServiceClient.prototype.writeRaw = function writeRaw(requestMessage, metadata, callback) {
  if (arguments.length === 2) {
    callback = arguments[1];
  }
  var client = grpc.unary(ProfileStoreService.WriteRaw, {
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

exports.ProfileStoreServiceClient = ProfileStoreServiceClient;

