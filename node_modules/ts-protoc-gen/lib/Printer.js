"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
var util_1 = require("./util");
var Printer = (function () {
    function Printer(indentLevel) {
        this.output = "";
        this.indentStr = util_1.generateIndent(indentLevel);
    }
    Printer.prototype.printLn = function (str) {
        this.output += this.indentStr + str + "\n";
    };
    Printer.prototype.print = function (str) {
        this.output += str;
    };
    Printer.prototype.printEmptyLn = function () {
        this.output += "\n";
    };
    Printer.prototype.printIndentedLn = function (str) {
        this.output += this.indentStr + "  " + str + "\n";
    };
    Printer.prototype.getOutput = function () {
        return this.output;
    };
    return Printer;
}());
exports.Printer = Printer;
//# sourceMappingURL=Printer.js.map