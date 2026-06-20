<script lang="ts">
  import brandMarkSvg from '$lib/assets/brand/brand-mark.svg?raw';
</script>

<span class="brand-mark" aria-hidden="true">
  {@html brandMarkSvg}
</span>

<style>
  .brand-mark :global(svg) {
    display: block;
    width: 18px;
    height: 18px;
  }
</style>
