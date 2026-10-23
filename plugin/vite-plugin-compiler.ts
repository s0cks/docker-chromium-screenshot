import fs from "fs";
import path from "path";
import type { Plugin } from "vite";
import { execSync } from "child_process";

interface GoPluginOptions {
  compilerPath: string;
}

export function goMarkdownPlugin(options: GoPluginOptions): Plugin {
  const binaryPath = path.resolve(options.compilerPath);
  const PREFIX = "go:";

  return {
    name: "vite-plugin-go-markdown",
    load(id) {
      if (!id.endsWith(".mdc")) return null;

      try {
        const fileContent = fs.readFileSync(id, "utf-8");

        const jsonOutput = execSync(`${binaryPath}`, {
          input: fileContent,
          encoding: "utf-8",
          maxBuffer: 1024 * 1024 * 10,
        });

        return `export default ${jsonOutput.trim()};`;
      } catch (error) {
        this.error(
          `Go Markdown Compiler failed on file ${filePath}:\n${error}`,
        );
      }
    },
  };
}
