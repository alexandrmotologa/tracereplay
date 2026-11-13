const puppeteer = require('puppeteer');
const path = require('path');
const fs = require('fs');
const { execSync } = require('child_process');

async function sleep(ms) {
  return new Promise((resolve) => setTimeout(resolve, ms));
}

async function capture() {
  const imagesDir = path.join(__dirname, '..', 'docs', 'images');
  const framesDir = path.join(__dirname, '..', 'docs', 'frames');
  fs.mkdirSync(imagesDir, { recursive: true });
  fs.mkdirSync(framesDir, { recursive: true });

  const browser = await puppeteer.launch({
    headless: true,
    args: ['--no-sandbox', '--disable-setuid-sandbox', '--window-size=1440,900'],
  });

  const page = await browser.newPage();
  await page.setViewport({ width: 1440, height: 900, deviceScaleFactor: 2 });

  console.log('Navigating to http://127.0.0.1:8085...');
  await page.goto('http://127.0.0.1:8085', { waitUntil: 'networkidle0', timeout: 15000 });
  await sleep(1500);

  // 1. Select the failed trajectory by filtering for "failed"
  console.log('Selecting failed trajectory...');
  await page.evaluate(() => {
    // Click "failed" filter tab
    const failedTab = Array.from(document.querySelectorAll('button')).find(
      (el) => el.textContent && el.textContent.trim().toLowerCase() === 'failed'
    );
    if (failedTab) failedTab.click();
  });
  await sleep(600);

  await page.evaluate(() => {
    // Click the failed trajectory item
    const failedItem = Array.from(document.querySelectorAll('aside div.group')).find(
      (el) => el.textContent && el.textContent.includes('steps')
    );
    if (failedItem) failedItem.click();
  });
  await sleep(1000);

  // Click Step 12 to show failure diagnostic and error banner
  await page.evaluate(() => {
    const step12Btn = Array.from(document.querySelectorAll('button')).find(
      (el) => el.getAttribute('title') && el.getAttribute('title').includes('Step 12')
    );
    if (step12Btn) step12Btn.click();
  });
  await sleep(800);

  // Capture Dashboard Desktop
  console.log('Capturing tracereplay_dashboard.png with failure & diagnostic...');
  await page.screenshot({
    path: path.join(imagesDir, 'tracereplay_dashboard.png'),
    fullPage: false,
  });

  // Reset filter back to "all"
  await page.evaluate(() => {
    const allTab = Array.from(document.querySelectorAll('button')).find(
      (el) => el.textContent && el.textContent.trim().toLowerCase() === 'all'
    );
    if (allTab) allTab.click();
  });
  await sleep(600);

  // 2. Capture DAG Flow
  console.log('Switching to DAG View...');
  await page.evaluate(() => {
    const dagBtn = Array.from(document.querySelectorAll('button')).find(
      (el) => el.textContent && el.textContent.includes('Execution DAG Flow')
    );
    if (dagBtn) dagBtn.click();
  });
  await sleep(1000);

  console.log('Capturing tracereplay_dag_flow.png...');
  await page.screenshot({
    path: path.join(imagesDir, 'tracereplay_dag_flow.png'),
    fullPage: false,
  });

  // Switch back to Timeline
  await page.evaluate(() => {
    const timelineBtn = Array.from(document.querySelectorAll('button')).find(
      (el) => el.textContent && el.textContent.includes('Timeline Scrubber')
    );
    if (timelineBtn) timelineBtn.click();
  });
  await sleep(800);

  // 3. Open Rewind Modal
  console.log('Opening Time-Travel Rewind Modal...');
  await page.evaluate(() => {
    const rewindBtn = Array.from(document.querySelectorAll('button')).find(
      (el) => el.textContent && el.textContent.includes('Rewind to Step 8') || el.textContent.includes('Rewind Here')
    );
    if (rewindBtn) rewindBtn.click();
  });
  await sleep(800);

  console.log('Capturing tracereplay_rewind.png...');
  await page.screenshot({
    path: path.join(imagesDir, 'tracereplay_rewind.png'),
    fullPage: false,
  });

  // Close Rewind Modal
  await page.evaluate(() => {
    const cancelBtn = Array.from(document.querySelectorAll('button')).find(
      (el) => el.textContent && (el.textContent.includes('Cancel') || el.textContent.includes('Close'))
    );
    if (cancelBtn) cancelBtn.click();
  });
  await sleep(600);

  // 4. Check Diff Selection & Open Diff Modal
  console.log('Selecting 2 trajectories for Diff...');
  await page.evaluate(() => {
    const checkboxes = document.querySelectorAll('input[type="checkbox"]');
    if (checkboxes.length >= 2) {
      if (!checkboxes[0].checked) checkboxes[0].click();
      if (!checkboxes[1].checked) checkboxes[1].click();
    }
  });
  await sleep(600);

  await page.evaluate(() => {
    const diffBtn = Array.from(document.querySelectorAll('button')).find(
      (el) => el.textContent && el.textContent.includes('Diff')
    );
    if (diffBtn && !diffBtn.disabled) diffBtn.click();
  });
  await sleep(1200);

  console.log('Capturing tracereplay_diff.png...');
  await page.screenshot({
    path: path.join(imagesDir, 'tracereplay_diff.png'),
    fullPage: false,
  });

  // Close Diff Modal
  await page.evaluate(() => {
    const closeBtn = Array.from(document.querySelectorAll('button')).find(
      (el) => el.textContent && el.textContent.includes('Close')
    );
    if (closeBtn) closeBtn.click();
  });
  await sleep(600);

  // 5. Capture Mobile Viewport (iPhone 15 Pro: 390x844)
  console.log('Capturing mobile view at 390x844...');
  await page.setViewport({ width: 390, height: 844, deviceScaleFactor: 2, isMobile: true, hasTouch: true });
  await sleep(800);

  // Open mobile drawer to showcase mobile trajectory selector
  await page.evaluate(() => {
    const menuBtn = document.querySelector('button[title*="Toggle trajectories"]');
    if (menuBtn) menuBtn.click();
  });
  await sleep(600);

  await page.screenshot({
    path: path.join(imagesDir, 'tracereplay_mobile_drawer.png'),
  });

  // Close drawer and show step inspector on mobile
  await page.evaluate(() => {
    const backdrop = document.querySelector('.fixed.inset-0.z-30');
    if (backdrop) backdrop.click();
  });
  await sleep(600);

  await page.screenshot({
    path: path.join(imagesDir, 'tracereplay_mobile.png'),
  });

  // 6. Record Animated Frames for Demo GIF
  console.log('Capturing animation frame sequence for GIF...');
  await page.setViewport({ width: 1200, height: 750, deviceScaleFactor: 1.5 });
  await sleep(800);

  let frameIdx = 0;
  const saveFrame = async () => {
    const fPath = path.join(framesDir, `frame_${String(frameIdx).padStart(3, '0')}.png`);
    await page.screenshot({ path: fPath });
    frameIdx++;
  };

  // Step through steps 0, 3, 6, 8, 11, 12
  const stepsToInspect = [0, 2, 4, 6, 8, 10, 12];
  for (const s of stepsToInspect) {
    await page.evaluate((idx) => {
      const btn = Array.from(document.querySelectorAll('button')).find(
        (el) => el.getAttribute('title') && el.getAttribute('title').includes(`Step ${idx}:`)
      );
      if (btn) btn.click();
    }, s);
    await sleep(250);
    for (let i = 0; i < 3; i++) {
      await saveFrame();
      await sleep(100);
    }
  }

  // Hover over AI diagnostic / rewind button
  for (let i = 0; i < 4; i++) {
    await saveFrame();
    await sleep(100);
  }

  // Open Rewind Modal
  await page.evaluate(() => {
    const rewindBtn = Array.from(document.querySelectorAll('button')).find(
      (el) => el.textContent && el.textContent.includes('Rewind & Fix') || el.textContent.includes('Rewind to Step')
    );
    if (rewindBtn) rewindBtn.click();
  });
  await sleep(300);
  for (let i = 0; i < 6; i++) {
    await saveFrame();
    await sleep(100);
  }

  // Close Rewind Modal
  await page.evaluate(() => {
    const cancelBtn = Array.from(document.querySelectorAll('button')).find(
      (el) => el.textContent && (el.textContent.includes('Cancel') || el.textContent.includes('Close'))
    );
    if (cancelBtn) cancelBtn.click();
  });
  await sleep(200);

  // Switch to DAG View
  await page.evaluate(() => {
    const dagBtn = Array.from(document.querySelectorAll('button')).find(
      (el) => el.textContent && el.textContent.includes('Execution DAG Flow')
    );
    if (dagBtn) dagBtn.click();
  });
  await sleep(300);
  for (let i = 0; i < 6; i++) {
    await saveFrame();
    await sleep(100);
  }

  // Open Diff Modal
  await page.evaluate(() => {
    const diffBtn = Array.from(document.querySelectorAll('button')).find(
      (el) => el.textContent && el.textContent.includes('Diff')
    );
    if (diffBtn && !diffBtn.disabled) diffBtn.click();
  });
  await sleep(400);
  for (let i = 0; i < 8; i++) {
    await saveFrame();
    await sleep(100);
  }

  await browser.close();
  console.log(`✓ Captured ${frameIdx} frames.`);

  // 7. Compile GIF with FFmpeg
  console.log('Encoding optimized animated GIF via FFmpeg...');
  const gifPath = path.join(imagesDir, 'tracereplay_demo.gif');
  const ffmpegCmd = `ffmpeg -y -framerate 6 -i "${framesDir}/frame_%03d.png" -vf "scale=900:-1:flags=lanczos,split[s0][s1];[s0]palettegen=max_colors=128:reserve_transparent=0[p];[s1][p]paletteuse=dither=bayer:bayer_scale=3" "${gifPath}"`;
  
  execSync(ffmpegCmd, { stdio: 'inherit' });
  console.log(`✓ Successfully encoded ${gifPath}`);

  // Cleanup frames
  fs.rmSync(framesDir, { recursive: true, force: true });
  console.log('✓ Cleaned up intermediate frames.');
}

capture().catch((err) => {
  console.error('Capture failed:', err);
  process.exit(1);
});
