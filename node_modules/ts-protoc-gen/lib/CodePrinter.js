"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
var util_1 = require("./util");
var CodePrinter = (function () {
    function CodePrinter(depth, printer) {
        this.depth = depth;
        this.printer = printer;
        this.indentation = util_1.generateIndent(1);
    }
    CodePrinter.prototype.indent = function () {
        this.depth++;
        return this;
    };
    CodePrinter.prototype.dedent = function () {
        this.depth--;
        return this;
    };
    CodePrinter.prototype.printLn = function (line) {
        this.printer.printLn(new Array(this.depth + 1).join(this.indentation) + line);
        return this;
    };
    CodePrinter.prototype.printEmptyLn = function () {
        this.printer.printEmptyLn();
        return this;
    };
    return CodePrinter;
}());
exports.CodePrinter = CodePrinter;
//# sourceMappingURL=CodePrinter.js.map