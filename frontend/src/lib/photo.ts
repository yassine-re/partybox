// Canvas re-encoding also removes EXIF metadata from the uploaded JPEG.
export async function compressPhoto(file: File): Promise<Blob> {
  if (!file.size || file.size > 25 * 1024 * 1024) {
    throw new Error("Choisis une photo de moins de 25 Mo.");
  }
  const url = URL.createObjectURL(file);
  const image = new Image();
  try {
    await new Promise<void>((resolve, reject) => {
      image.onload = () => resolve();
      image.onerror = () => reject(new Error("Photo illisible. Essaie une photo JPEG, PNG ou WebP."));
      image.src = url;
    });
    const scale = Math.min(1, 1280 / Math.max(image.naturalWidth, image.naturalHeight));
    const canvas = document.createElement("canvas");
    canvas.width = Math.max(1, Math.round(image.naturalWidth * scale));
    canvas.height = Math.max(1, Math.round(image.naturalHeight * scale));
    const ctx = canvas.getContext("2d");
    if (!ctx) throw new Error("Ce navigateur ne peut pas préparer la photo.");
    ctx.fillStyle = "white";
    ctx.fillRect(0, 0, canvas.width, canvas.height);
    ctx.drawImage(image, 0, 0, canvas.width, canvas.height);
    const blob = await new Promise<Blob>((resolve, reject) => {
      canvas.toBlob((value) => value ? resolve(value) : reject(new Error("Compression impossible.")), "image/jpeg", 0.8);
    });
    if (blob.size > 5 * 1024 * 1024) throw new Error("La photo reste trop volumineuse. Essaie une image plus petite.");
    return blob;
  } finally {
    image.src = "";
    URL.revokeObjectURL(url);
  }
}
