import {
  forceSimulation,
  forceLink,
  forceManyBody,
  forceCenter,
  forceCollide,
} from "d3-force";

/**
 * Computes deterministic static coordinates for a network topology.
 * @param {Array} nodes - Raw node entities.
 * @param {Array} edges - Source/Target relationships.
 * @param {Object} opts - Layout configurations.
 */
export function calculateStaticNetworkLayout(nodes, edges, opts = {}) {
  const width = opts.width ?? 800;
  const height = opts.height ?? 600;
  const distance = opts.distance ?? 100;
  const iterations = opts.iterations ?? 300;

  // Deep clone data to avoid mutating original source properties across render runs
  const layoutNodes = nodes.map((n) => ({ ...n }));
  const layoutEdges = edges.map((e) => ({ ...e }));

  // Initialize and instantly arrest the live animation loops
  const simulation = forceSimulation(layoutNodes)
    .force(
      "link",
      forceLink(layoutEdges)
        .id((d) => d.id)
        .distance(distance),
    )
    .force("charge", forceManyBody().strength(-350))
    .force("center", forceCenter(width / 2, height / 2))
    .force("collision", forceCollide().radius(28))
    .stop();

  // Run finite structural ticks synchronously to find the stability anchor point
  for (let i = 0; i < iterations; ++i) simulation.tick();

  return { nodes: layoutNodes, edges: layoutEdges };
}
