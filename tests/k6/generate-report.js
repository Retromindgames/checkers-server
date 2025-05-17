const reporter = require('k6-html-reporter');
const path = require('path');

const options = {
        jsonFile: path.resolve(__dirname, 'summary.json'),
        output: '.',
    };

reporter.generateSummaryReport(options);