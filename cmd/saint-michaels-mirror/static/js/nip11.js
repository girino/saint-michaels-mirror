/*
Copyright (c) 2025 Girino Vey.

This software is licensed under Girino's Anarchist License (GAL).
See LICENSE file for full license text.
License available at: https://license.girino.org/

NIP-11 JSON loader for Espelho de São Miguel web interface.
*/

// NIP-11 JSON loader
document.getElementById('nip11-details').addEventListener('toggle', async function(e){
  if(!this.open) return;
  const pre = document.getElementById('nip11');
  pre.textContent = 'loading...';
  try{
    const resp = await fetch('/', { headers: { 'Accept': 'application/nostr+json' } });
    if(!resp.ok) throw new Error('failed to fetch');
    const data = await resp.json();
    pre.textContent = JSON.stringify(data, null, 2);
  }catch(err){
    pre.textContent = 'failed to load NIP-11: '+err;
  }
});
