"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
var Printer_1 = require("../Printer");
var util_1 = require("../util");
function printEnum(enumDescriptor, indentLevel) {
    var printer = new Printer_1.Printer(indentLevel);
    var enumInterfaceName = enumDescriptor.getName() + "Map";
    printer.printEmptyLn();
    printer.printLn("export interface " + enumInterfaceName + " {");
    enumDescriptor.getValueList().forEach(function (value) {
        var valueName = value.getName() || util_1.throwError("Missing value name");
        printer.printIndentedLn(valueName.toUpperCase() + ": " + value.getNumber() + ";");
    });
    printer.printLn("}");
    printer.printEmptyLn();
    printer.printLn("export const " + enumDescriptor.getName() + ": " + enumInterfaceName + ";");
    return printer.getOutput();
}
exports.printEnum = printEnum;
//# sourceMappingURL=enum.js.map