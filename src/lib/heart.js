const HEART_PIXEL_MASK = [
  [0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0],
  [0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0],
  [0, 0, 0, 1, 1, 0, 0, 0, 0, 1, 1, 0, 0, 0, 0, 0],
  [0, 0, 1, 1, 1, 1, 0, 0, 1, 1, 1, 1, 0, 0, 0, 0],
  [0, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 0, 0, 0],
  [1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 0, 0],
  [1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 0],
  [1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 0],
  [1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 0],
  [0, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 0, 0],
  [0, 0, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 0, 0, 0],
  [0, 0, 0, 1, 1, 1, 1, 1, 1, 1, 1, 1, 0, 0, 0, 0],
  [0, 0, 0, 0, 1, 1, 1, 1, 1, 1, 1, 0, 0, 0, 0, 0],
  [0, 0, 0, 0, 0, 1, 1, 1, 1, 1, 0, 0, 0, 0, 0, 0],
  [0, 0, 0, 0, 0, 0, 1, 1, 1, 0, 0, 0, 0, 0, 0, 0],
  [0, 0, 0, 0, 0, 0, 0, 1, 0, 0, 0, 0, 0, 0, 0, 0],
];

export function generateHeartHashSVG(hexHash, color = "#e91e63") {
  if (!/^[0-9a-fA-F]{64}$/.test(hexHash)) {
    throw new Error("Invalid 256-bit Hex Hash");
  }

  let binaryString = "";
  for (let char of hexHash) {
    binaryString += parseInt(char, 16).toString(2).padStart(4, "0");
  }

  const cellSize = 6;
  let maskRects = "";
  let dataRects = "";
  let bitIndex = 0;

  for (let r = 0; r < 16; r++) {
    for (let c = 0; c < 16; c++) {
      const x = c * cellSize;
      const y = r * cellSize;

      if (HEART_PIXEL_MASK[r][c] === 1) {
        const fill = binaryString[bitIndex] === "1" ? color : "transparent";
        dataRects += `<rect x="${x}" y="${y}" width="${cellSize}" height="${cellSize}" fill="${fill}" />\n`;
        bitIndex++;
      }
    }
  }

  return `<svg xmlns="http://w3.org" viewBox="0 0 100 100">
    <g>
      ${dataRects.trim()}
    </g>
  </svg>`;
}

/**
 * Decodes a generated SVG element or string back into the original 256-bit Hex Hash.
 * @param {SVGElement|string} svgInput - The SVG element DOM node or raw SVG string.
 * @returns {string} 64-character Hex Hash
 */
function decodeHeartHashSVG(svgInput) {
  let doc = svgInput;

  if (typeof svgInput === "string") {
    const parser = new DOMParser();
    doc = parser.parseFromString(svgInput, "image/svg+xml").documentElement;
  }

  const bits = new Array(256).fill("0");

  const rects = doc.querySelectorAll("g[clip-path] rect[data-idx]");

  rects.forEach((rect) => {
    const idx = parseInt(rect.getAttribute("data-idx"), 10);
    if (idx >= 0 && idx < 256) {
      bits[idx] = "1";
    }
  });

  const binaryString = bits.join("");

  let hexHash = "";
  for (let i = 0; i < binaryString.length; i += 4) {
    const chunk = binaryString.substring(i, i + 4);
    hexHash += parseInt(chunk, 2).toString(16);
  }

  return hexHash;
}
