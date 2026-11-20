<script lang="ts">
  import { pickRequest, closePick } from '../lib/filePicker';
  import FileBrowserDialog from './FileBrowserDialog.svelte';
</script>

{#if $pickRequest}
  {#key $pickRequest}
    <FileBrowserDialog kind={$pickRequest.kind} scope={$pickRequest.scope} title={$pickRequest.title} onDone={closePick} />
  {/key}
{/if}
