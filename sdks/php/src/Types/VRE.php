<?php

declare(strict_types=1);

namespace Linktor\Types;

/**
 * VRE render response
 */
class VRERenderResponse
{
    public string $imageBase64 = '';
    public string $caption = '';
    public int $width = 0;
    public int $height = 0;
    public string $format = '';
    public int $renderTimeMs = 0;
    public ?int $sizeBytes = null;
    public ?bool $cacheHit = null;

    public static function fromArray(array $data): self
    {
        $instance = new self();
        $instance->imageBase64 = $data['image_base64'] ?? $data['imageBase64'] ?? '';
        $instance->caption = $data['caption'] ?? '';
        $instance->width = $data['width'] ?? 0;
        $instance->height = $data['height'] ?? 0;
        $instance->format = $data['format'] ?? '';
        $instance->renderTimeMs = $data['render_time_ms'] ?? $data['renderTimeMs'] ?? 0;
        $instance->sizeBytes = $data['size_bytes'] ?? $data['sizeBytes'] ?? null;
        $instance->cacheHit = $data['cache_hit'] ?? $data['cacheHit'] ?? null;
        return $instance;
    }
}

/**
 * VRE render and send response
 */
class VRERenderAndSendResponse
{
    public string $messageId = '';
    public string $imageUrl = '';
    public string $caption = '';

    public static function fromArray(array $data): self
    {
        $instance = new self();
        $instance->messageId = $data['message_id'] ?? $data['messageId'] ?? '';
        $instance->imageUrl = $data['image_url'] ?? $data['imageUrl'] ?? '';
        $instance->caption = $data['caption'] ?? '';
        return $instance;
    }
}

/**
 * VRE template definition
 */
class VRETemplate
{
    public string $id = '';
    public string $name = '';
    public string $description = '';
    public array $schema = [];

    public static function fromArray(array $data): self
    {
        $instance = new self();
        $instance->id = $data['id'] ?? '';
        $instance->name = $data['name'] ?? '';
        $instance->description = $data['description'] ?? '';
        $instance->schema = $data['schema'] ?? [];
        return $instance;
    }
}

/**
 * VRE list templates response
 */
class VREListTemplatesResponse
{
    /** @var VRETemplate[] */
    public array $templates = [];

    public static function fromArray(array $data): self
    {
        $instance = new self();
        $instance->templates = array_map(
            fn($t) => VRETemplate::fromArray($t),
            $data['templates'] ?? []
        );
        return $instance;
    }
}

/**
 * VRE preview response
 */
class VREPreviewResponse
{
    public string $imageBase64 = '';
    public int $width = 0;
    public int $height = 0;

    public static function fromArray(array $data): self
    {
        $instance = new self();
        $instance->imageBase64 = $data['image_base64'] ?? $data['imageBase64'] ?? '';
        $instance->width = $data['width'] ?? 0;
        $instance->height = $data['height'] ?? 0;
        return $instance;
    }
}
