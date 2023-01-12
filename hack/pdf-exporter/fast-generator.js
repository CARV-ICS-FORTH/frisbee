'use strict';

const puppeteer = require('puppeteer');

// URL to load should be passed as first parameter
const url = process.argv[2];
// Username and password (with colon separator) should be second parameter
const auth_string = process.argv[3];
// Output file name should be third parameter
const outfile = process.argv[4];

// TODO: Output an error message if number of arguments is not right or arguments are invalid

// Set the browser width in pixels. The paper size will be calculated on the basus of 96dpi,
// so 1200 corresponds to 12.5".
const width_px = 1632;
// Note that to get an actual paper size, e.g. Letter, you will want to *not* simply set the pixel
// size here, since that would lead to a "mobile-sized" screen (816px), and mess up the rendering.
// Instead, set e.g. double the size here (1632px), and call page.pdf() with format: 'Letter' and
// scale = 0.5.

// Generate authorization header for basic auth
const auth_header = 'Basic ' + new Buffer.from(auth_string).toString('base64');

(async () => {
    // const browser = await puppeteer.launch();
    const browser = await puppeteer.launch({args: ['--no-sandbox', '--disable-setuid-sandbox']});
    const page = await browser.newPage();

    // Set basic auth headers
    await page.setExtraHTTPHeaders({'Authorization': auth_header});

    // Increase timeout from the default of 30 seconds to 120 seconds, to allow for slow-loading panels
    await page.setDefaultNavigationTimeout(120000);

    // Increasing the deviceScaleFactor gets a higher-resolution image. The width should be set to
    // the same value as in page.pdf() below. The height is not important
    await page.setViewport({
        width: width_px,
        height: 800,
        deviceScaleFactor: 1,
        isMobile: false
    })

    // Wait until all network connections are closed (and none are opened withing 0.5s).
    // In some cases it may be appropriate to change this to {waitUntil: 'networkidle2'},
    // which stops when there are only 2 or fewer connections remaining.
    await page.goto(url, {waitUntil: 'networkidle0'});

    // Hide all panel description (top-left "i") pop-up handles and, all panel resize handles
    // Annoyingly, it seems you can't concatenate the two object collections into one
    await page.evaluate(() => {
        let infoCorners = document.getElementsByClassName('panel-info-corner');
        for (el of infoCorners) {
            el.hidden = true;
        }
        ;
        let resizeHandles = document.getElementsByClassName('react-resizable-handle');
        for (el of resizeHandles) {
            el.hidden = true;
        }
        ;
    });

    // Get the height of the main canvas, and add a margin
    var height_px = await page.evaluate(() => {
        document.querySelector('.react-grid-layout').style.height = 'auto';
        return document.getElementsByClassName('react-grid-layout')[0].getBoundingClientRect().bottom;
    }) + 20;

    await page.pdf({
        path: outfile,
        width: width_px + 'px',
        height: height_px + 'px',
        // format: 'Letter',  // <-- see note above for generating "paper-sized" outputs
        scale: 1,
        displayHeaderFooter: false,
        printBackground: true,
        margin: {
            top: 0,
            right: 0,
            bottom: 0,
            left: 0,
        },
    });

    await browser.close();
})();
