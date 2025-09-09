import fs from 'fs';
import sharp from 'sharp';
import png2icons from 'png2icons';

async function convertSvgToPng() {
    const windowsDir = '../build/windows';
    const svgPath = `${windowsDir}/ghost.svg`;
    const pngPath = `${windowsDir}/temp-256.png`;
    const icoPath = `${windowsDir}/icon.ico`;  // This will overwrite the existing icon.ico

    try {
        // Convert SVG to PNG using sharp
        await sharp(svgPath)
            .resize(256, 256)
            .png()
            .toFile(pngPath);
        
        console.log('Generated PNG file');

        // Read the PNG file
        const input = fs.readFileSync(pngPath);
        
        // Convert PNG to ICO with multiple sizes (16, 32, 48, 256)
        const output = png2icons.createICO(input, png2icons.BILINEAR, 0, false, true);
        
        if (output) {
            // Save the icon file, overwriting any existing icon.ico
            fs.writeFileSync(icoPath, output);
            console.log('Successfully created icon.ico in build/windows directory');

            // Clean up temporary files
            fs.unlinkSync(pngPath);
        } else {
            console.error('Failed to create ICO file');
        }

    } catch (error) {
        console.error('Error:', error);
        console.error('Error details:', error.message);
    }
}

convertSvgToPng(); 