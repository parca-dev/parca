// package: parca.scrape.v1alpha1
// file: parca/scrape/v1alpha1/scrape.proto

var parca_scrape_v1alpha1_scrape_pb = require('../../../parca/scrape/v1alpha1/scrape_pb');
var grpc = require('@improbable-eng/grpc-web').grpc;

var ScrapeService = (function () {
  function ScrapeService() {}
  ScrapeService.serviceName = 'parca.scrape.v1alpha1.ScrapeService';
  return ScrapeService;
})();

ScrapeService.Targets = {
  methodName: 'Targets',
  service: ScrapeService,
  requestStream: false,
  responseStream: false,
  requestType: parca_scrape_v1alpha1_scrape_pb.TargetsRequest,
  responseType: parca_scrape_v1alpha1_scrape_pb.TargetsResponse,
};

exports.ScrapeService = ScrapeService;

function ScrapeServiceClient(serviceHost, options) {
  this.serviceHost = serviceHost;
  this.options = options || {};
}

ScrapeServiceClient.prototype.targets = function targets(requestMessage, metadata, callback) {
  if (arguments.length === 2) {
    callback = arguments[1];
  }
  var client = grpc.unary(ScrapeService.Targets, {
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

exports.ScrapeServiceClient = ScrapeServiceClient;
