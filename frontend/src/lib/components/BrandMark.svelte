<script lang="ts">
  import brandMarkSvg from '$lib/assets/brand/junimo-hut-mark.svg?raw';
</script>

<span class="brand-mark" aria-hidden="true">
  {@html brandMarkSvg}
</span>

<style>
  .brand-mark :global(svg) {
    display: block;
    width: 1.25rem;
    height: 1.25rem;
    shape-rendering: crispEdges;
    image-rendering: pixelated;
    image-rendering: -moz-crisp-edges;
  }
</style>
