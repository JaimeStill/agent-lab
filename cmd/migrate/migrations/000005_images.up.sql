CREATE TABLE images (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  document_id UUID NOT NULL REFERENCES documents(id) ON DELETE CASCADE,
  page_number INTEGER NOT NULL,
  format TEXT NOT NULL,
  dpi INTEGER NOT NULL,
  quality INTEGER,
  brightness INTEGER,
  contrast INTEGER,
  saturation INTEGER,
  rotation INTEGER,
  background TEXT,
  storage_key TEXT NOT NULL UNIQUE,
  size_bytes BIGINT NOT NULL,
  created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),

  UNIQUE(document_id, page_number, format, dpi, quality,
         brightness, contrast, saturation, rotation, background)
);

CREATE INDEX idx_images_document_id ON images(document_id);
CREATE INDEX idx_images_created_at ON images(created_at DESC);
