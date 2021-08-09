"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
var util_1 = require("../util");
exports.MESSAGE_TYPE = 11;
exports.BYTES_TYPE = 12;
exports.ENUM_TYPE = 14;
var TypeNumToTypeString = {};
TypeNumToTypeString[1] = "number";
TypeNumToTypeString[2] = "number";
TypeNumToTypeString[3] = "number";
TypeNumToTypeString[4] = "number";
TypeNumToTypeString[5] = "number";
TypeNumToTypeString[6] = "number";
TypeNumToTypeString[7] = "number";
TypeNumToTypeString[8] = "boolean";
TypeNumToTypeString[9] = "string";
TypeNumToTypeString[10] = "Object";
TypeNumToTypeString[exports.MESSAGE_TYPE] = "Object";
TypeNumToTypeString[exports.BYTES_TYPE] = "Uint8Array";
TypeNumToTypeString[13] = "number";
TypeNumToTypeString[exports.ENUM_TYPE] = "number";
TypeNumToTypeString[15] = "number";
TypeNumToTypeString[16] = "number";
TypeNumToTypeString[17] = "number";
TypeNumToTypeString[18] = "number";
function getTypeName(fieldTypeNum) {
    return TypeNumToTypeString[fieldTypeNum];
}
exports.getTypeName = getTypeName;
function getFieldType(type, typeName, currentFileName, exportMap) {
    if (type === exports.MESSAGE_TYPE) {
        if (!typeName)
            return util_1.throwError("Type was Message, but typeName is not set");
        var fromExport = exportMap.getMessage(typeName);
        if (!fromExport) {
            return util_1.throwError("Could not getFieldType for message: " + typeName);
        }
        var withinNamespace = util_1.withinNamespaceFromExportEntry(typeName, fromExport);
        if (fromExport.fileName === currentFileName) {
            return withinNamespace;
        }
        else {
            return util_1.filePathToPseudoNamespace(fromExport.fileName) + "." + withinNamespace;
        }
    }
    else if (type === exports.ENUM_TYPE) {
        if (!typeName)
            return util_1.throwError("Type was Enum, but typeName is not set");
        var fromExport = exportMap.getEnum(typeName);
        if (!fromExport) {
            return util_1.throwError("Could not getFieldType for enum: " + typeName);
        }
        var withinNamespace = util_1.withinNamespaceFromExportEntry(typeName, fromExport);
        if (fromExport.fileName === currentFileName) {
            return withinNamespace + "Map";
        }
        else {
            return util_1.filePathToPseudoNamespace(fromExport.fileName) + "." + withinNamespace;
        }
    }
    else {
        return TypeNumToTypeString[type];
    }
}
exports.getFieldType = getFieldType;
//# sourceMappingURL=FieldTypes.js.map