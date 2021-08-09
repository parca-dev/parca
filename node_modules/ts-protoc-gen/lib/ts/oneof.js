"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
var Printer_1 = require("../Printer");
var util_1 = require("../util");
function printOneOfDecl(oneOfDecl, oneOfFields, indentLevel) {
    var printer = new Printer_1.Printer(indentLevel);
    printer.printEmptyLn();
    var oneOfDeclName = oneOfDecl.getName() || util_1.throwError("Missing one_of name");
    printer.printLn("export enum " + util_1.oneOfName(oneOfDeclName) + "Case {");
    printer.printIndentedLn(oneOfDeclName.toUpperCase() + "_NOT_SET = 0,");
    oneOfFields.forEach(function (field) {
        var fieldName = field.getName() || util_1.throwError("Missing field name");
        printer.printIndentedLn(fieldName.toUpperCase() + " = " + field.getNumber() + ",");
    });
    printer.printLn("}");
    return printer.output;
}
exports.printOneOfDecl = printOneOfDecl;
//# sourceMappingURL=oneof.js.map