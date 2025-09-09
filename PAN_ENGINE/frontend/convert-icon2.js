import fs from 'fs';
import { promisify } from 'util';
import { exec } from 'child_process';
import png2icons from 'png2icons';

const execAsync = promisify(exec);

// First convert SVG to PNG using Inkscape (which handles SVG conversion better)
async function convertSvgToPng() {
    try {
        // Create PNG in different sizes
        const sizes = [16, 32, 48, 64, 128, 256];
        for (const size of sizes) {
            await execAsync(`inkscape ../build/windows/ghost.svg --export-type=png -w ${size} -h ${size} --export-filename=icon-${size}.png`);
        }
        
        // Read the largest PNG file
        const pngBuffer = fs.readFileSync('icon-256.png');
        
        // Convert to ICO
        const icoData = png2icons.createICO(pngBuffer, png2icons.BILINEAR, 0, false, true);
        
        // Write the ICO file
        fs.writeFileSync('../build/windows/icon.ico', icoData);
        console.log('Icon converted successfully!');
        
        // Clean up PNG files
        sizes.forEach(size => {
            fs.unlinkSync(`icon-${size}.png`);
        });
    } catch (err) {
        console.error('Error:', err);
    }
}

convertSvgToPng(); 