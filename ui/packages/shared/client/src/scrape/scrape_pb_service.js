// package: parca.scrape
// file: scrape/scrape.proto

var scrape_scrape_pb = require("../scrape/scrape_pb");
var grpc = require("@improbable-eng/grpc-web").grpc;

var Scrape = (function () {
  function Scrape() {}
  Scrape.serviceName = "parca.scrape.Scrape";
  return Scrape;
}());

Scrape.Targets = {
  methodName: "Targets",
  service: Scrape,
  requestStream: false,
  responseStream: false,
  requestType: scrape_scrape_pb.TargetsRequest,
  responseType: scrape_scrape_pb.TargetsResponse
};

exports.Scrape = Scrape;

function ScrapeClient(serviceHost, options) {
  this.serviceHost = serviceHost;
  this.options = options || {};
}

ScrapeClient.prototype.targets = function targets(requestMessage, metadata, callback) {
  if (arguments.length === 2) {
    callback = arguments[1];
  }
  var client = grpc.unary(Scrape.Targets, {
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

exports.ScrapeClient = ScrapeClient;

