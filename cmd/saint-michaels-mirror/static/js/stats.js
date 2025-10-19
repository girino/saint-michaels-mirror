// Stats loader
document.getElementById('stats-details').addEventListener('toggle', async function(e){
  if(!this.open) return;
  const pre = document.getElementById('stats');
  pre.textContent = 'loading...';
  try{
    const resp = await fetch('/api/v1/stats');
    if(!resp.ok) throw new Error('failed to fetch');
    const data = await resp.json();
    pre.textContent = JSON.stringify(data, null, 2);
  }catch(err){
    pre.textContent = 'failed to load stats: '+err;
  }
});
