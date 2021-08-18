// package: parca.profilestore
// file: profilestore/profilestore.proto

var profilestore_profilestore_pb = require("../profilestore/profilestore_pb");
var grpc = require("@improbable-eng/grpc-web").grpc;

var ProfileStore = (function () {
  function ProfileStore() {}
  ProfileStore.serviceName = "parca.profilestore.ProfileStore";
  return ProfileStore;
}());

ProfileStore.WriteRaw = {
  methodName: "WriteRaw",
  service: ProfileStore,
  requestStream: false,
  responseStream: false,
  requestType: profilestore_profilestore_pb.WriteRawRequest,
  responseType: profilestore_profilestore_pb.WriteRawResponse
};

exports.ProfileStore = ProfileStore;

function ProfileStoreClient(serviceHost, options) {
  this.serviceHost = serviceHost;
  this.options = options || {};
}

ProfileStoreClient.prototype.writeRaw = function writeRaw(requestMessage, metadata, callback) {
  if (arguments.length === 2) {
    callback = arguments[1];
  }
  var client = grpc.unary(ProfileStore.WriteRaw, {
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

exports.ProfileStoreClient = ProfileStoreClient;

