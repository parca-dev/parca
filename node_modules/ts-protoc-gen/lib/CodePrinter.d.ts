import { Printer } from "./Printer";
export declare class CodePrinter {
    private depth;
    private printer;
    private indentation;
    constructor(depth: number, printer: Printer);
    indent(): this;
    dedent(): this;
    printLn(line: string): this;
    printEmptyLn(): this;
}
