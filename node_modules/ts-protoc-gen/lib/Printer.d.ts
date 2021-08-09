export declare class Printer {
    indentStr: string;
    output: string;
    constructor(indentLevel: number);
    printLn(str: string): void;
    print(str: string): void;
    printEmptyLn(): void;
    printIndentedLn(str: string): void;
    getOutput(): string;
}
