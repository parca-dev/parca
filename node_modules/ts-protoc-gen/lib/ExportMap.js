"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
var util_1 = require("./util");
var ExportMap = (function () {
    function ExportMap() {
        this.messageMap = {};
        this.enumMap = {};
    }
    ExportMap.prototype.exportNested = function (scope, fileDescriptor, message) {
        var _this = this;
        var messageName = message.getName() || util_1.throwError("Missing message name for message. Scope: " + scope);
        var messageOptions = message.getOptions();
        var mapFieldOptions = undefined;
        if (messageOptions && messageOptions.getMapEntry()) {
            var keyType = message.getFieldList()[0].getType() || util_1.throwError("Missing map key type for message. Scope: " + scope + " Message: " + messageName);
            var keyTypeName = message.getFieldList()[0].getTypeName();
            var valueType = message.getFieldList()[1].getType() || util_1.throwError("Missing map value type for message. Scope: " + scope + " Message: " + messageName);
            var valueTypeName = message.getFieldList()[1].getTypeName();
            mapFieldOptions = {
                key: [
                    keyType,
                    keyTypeName ? keyTypeName.slice(1) : null,
                ],
                value: [
                    valueType,
                    valueTypeName ? valueTypeName.slice(1) : null,
                ],
            };
        }
        var pkg = fileDescriptor.getPackage() || "";
        var fileName = fileDescriptor.getName() || util_1.throwError("Missing file name for message. Scope: " + scope + " Message: " + messageName);
        var messageEntry = {
            pkg: pkg,
            fileName: fileName,
            messageOptions: messageOptions,
            mapFieldOptions: mapFieldOptions,
        };
        var packagePrefix = scope ? scope + "." : "";
        var entryName = "" + packagePrefix + messageName;
        this.messageMap[entryName] = messageEntry;
        message.getNestedTypeList().forEach(function (nested) {
            _this.exportNested("" + packagePrefix + messageName, fileDescriptor, nested);
        });
        message.getEnumTypeList().forEach(function (enumType) {
            var enumName = enumType.getName();
            var identifier = "" + packagePrefix + messageName + "." + enumName;
            _this.enumMap[identifier] = {
                pkg: pkg,
                fileName: fileName,
                enumOptions: enumType.getOptions(),
            };
        });
    };
    ExportMap.prototype.addFileDescriptor = function (fileDescriptor) {
        var _this = this;
        var scope = fileDescriptor.getPackage() || "";
        fileDescriptor.getMessageTypeList().forEach(function (messageType) {
            _this.exportNested(scope, fileDescriptor, messageType);
        });
        fileDescriptor.getEnumTypeList().forEach(function (enumType) {
            var packagePrefix = scope ? scope + "." : "";
            var enumName = enumType.getName();
            _this.enumMap[packagePrefix + enumName] = {
                pkg: scope,
                fileName: fileDescriptor.getName() || util_1.throwError("Missing file descriptor name for enum. Scope: " + scope + " Enum: " + enumName),
                enumOptions: enumType.getOptions(),
            };
        });
    };
    ExportMap.prototype.getMessage = function (str) {
        return this.messageMap[str];
    };
    ExportMap.prototype.getEnum = function (str) {
        return this.enumMap[str];
    };
    return ExportMap;
}());
exports.ExportMap = ExportMap;
//# sourceMappingURL=ExportMap.js.map