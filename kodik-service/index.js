import express from 'express';
import cors from 'cors';
import { VideoLinks } from 'kodikwrapper';

const app = express();
const PORT = process.env.PORT || 3333;

app.use(cors());
app.use(express.json());

app.get('/', (req, res) => {
  res.json({ status: 'ok', service: 'kodik-service' });
});

// POST /api/parse
// Body: { "link": "//kodikplayer.com/seria/723187/HASH/720p" }
// Ответ: { video_url, quality, all_qualities }
app.post('/api/parse', async (req, res) => {
  const { link } = req.body || {};
  if (!link) {
    return res.status(400).json({ error: 'Параметр link обязателен' });
  }

  // Простая валидация — что это действительно ссылка Kodik
  if (!link.includes('kodikplayer.com') && !link.includes('kodik.info')) {
    return res.status(400).json({ error: 'Ссылка должна быть с kodikplayer.com или kodik.info' });
  }

  if (!link.includes('/seria/') && !link.includes('/video/')) {
    return res.status(400).json({
      error: 'Нужна ссылка на конкретную серию (/seria/... или /video/...), а не на сериал (/serial/...)',
    });
  }

  try {
    console.log('[parse] Ссылка:', link);

    const parsedLink = await VideoLinks.parseLink({
      link,
      extended: true,
    });

    console.log('[parse] playerSingleUrl:', parsedLink.ex?.playerSingleUrl);

    // Получаем актуальный endpoint (Kodik его периодически меняет)
    let endpoint;
    if (parsedLink.ex?.playerSingleUrl) {
      endpoint = await VideoLinks.getActualVideoInfoEndpoint(parsedLink.ex.playerSingleUrl);
      console.log('[parse] Актуальный endpoint:', endpoint);
    }

    const links = await VideoLinks.getLinks({
      link,
      videoInfoEndpoint: endpoint,
    });

    const best = links['720']?.[0] || links['480']?.[0] || links['360']?.[0];
    if (!best) {
      return res.status(404).json({ error: 'Видео не найдено' });
    }

    const videoUrl = best.src.startsWith('//') ? 'https:' + best.src : best.src;

    res.json({
      video_url: videoUrl,
      quality: Object.keys(links).find(k => links[k][0] === best) || '720',
      all_qualities: Object.keys(links),
      all_links: links,
    });
  } catch (e) {
    console.error('[parse] Ошибка:', e.message);
    res.status(500).json({ error: e.message });
  }
});

app.listen(PORT, '0.0.0.0', () => {
  console.log(`✓ Kodik-service запущен: http://0.0.0.0:${PORT}`);
});