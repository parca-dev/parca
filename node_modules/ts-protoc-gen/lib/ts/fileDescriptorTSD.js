"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
var util_1 = require("../util");
var Printer_1 = require("../Printer");
var WellKnown_1 = require("../WellKnown");
var message_1 = require("./message");
var enum_1 = require("./enum");
var extensions_1 = require("./extensions");
function printFileDescriptorTSD(fileDescriptor, exportMap) {
    var fileName = fileDescriptor.getName() || util_1.throwError("Missing file descriptor name");
    var packageName = fileDescriptor.getPackage();
    var printer = new Printer_1.Printer(0);
    printer.printLn("// package: " + packageName);
    printer.printLn("// file: " + fileDescriptor.getName());
    var upToRoot = util_1.getPathToRoot(fileName);
    printer.printEmptyLn();
    printer.printLn("import * as jspb from \"google-protobuf\";");
    fileDescriptor.getDependencyList().forEach(function (dependency) {
        var pseudoNamespace = util_1.filePathToPseudoNamespace(dependency);
        if (dependency in WellKnown_1.WellKnownTypesMap) {
            printer.printLn("import * as " + pseudoNamespace + " from \"" + WellKnown_1.WellKnownTypesMap[dependency] + "\";");
        }
        else {
            var filePath = util_1.replaceProtoSuffix(dependency);
            printer.printLn("import * as " + pseudoNamespace + " from \"" + upToRoot + filePath + "\";");
        }
    });
    fileDescriptor.getMessageTypeList().forEach(function (enumType) {
        printer.print(message_1.printMessage(fileName, exportMap, enumType, 0, fileDescriptor));
    });
    fileDescriptor.getExtensionList().forEach(function (extension) {
        printer.print(extensions_1.printExtension(fileName, exportMap, extension, 0));
    });
    fileDescriptor.getEnumTypeList().forEach(function (enumType) {
        printer.print(enum_1.printEnum(enumType, 0));
    });
    printer.printEmptyLn();
    return printer.getOutput();
}
exports.printFileDescriptorTSD = printFileDescriptorTSD;
//# sourceMappingURL=fileDescriptorTSD.js.map