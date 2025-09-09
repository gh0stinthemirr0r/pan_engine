import icongen from 'icon-gen';

icongen('../build/windows/ghost.svg', '../build/windows', {
    report: true,
    ico: {
        name: 'icon',
        sizes: [16, 24, 32, 48, 64, 128, 256]
    }
}).then((results) => {
    console.log('Icon generated successfully:', results);
}).catch((err) => {
    console.error('Error generating icon:', err);
}); 