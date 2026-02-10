# Community

The imagor ecosystem includes several community-contributed projects that extend and integrate with imagor:

- **[cshum/imagor-studio](https://github.com/cshum/imagor-studio)** - Image gallery and live editing web application for creators
- **[cshum/imagorvideo](https://github.com/cshum/imagorvideo)** - imagor video thumbnail server in Go and ffmpeg C bindings
- **[sandstorm/laravel-imagor](https://github.com/sandstorm/laravel-imagor)** - Laravel integration for imagor
- **[codedoga/imagor-toy](https://github.com/codedoga/imagor-toy)** - A ReactJS based app to play with Imagor

## Ecosystem Projects

### imagor-studio

**[cshum/imagor-studio](https://github.com/cshum/imagor-studio)** is a comprehensive image gallery and live editing web application designed for creators and content managers.

**Features:**
- **Live image editing** with real-time preview
- **Gallery management** for organizing images
- **Batch processing** capabilities
- **Integration** with imagor backend
- **User-friendly interface** for non-technical users

**Use Cases:**
- Content management systems
- Digital asset management
- Creative workflows
- Image optimization pipelines

### imagorvideo

**[cshum/imagorvideo](https://github.com/cshum/imagorvideo)** extends imagor's capabilities to video processing, providing thumbnail generation and basic video operations.

**Features:**
- **Video thumbnail generation** from any frame
- **FFmpeg integration** through Go bindings
- **Multiple video formats** support
- **Consistent API** with imagor
- **High performance** video processing

**Use Cases:**
- Video streaming platforms
- Media management systems
- Video preview generation
- Thumbnail creation workflows

### Laravel Integration

**[sandstorm/laravel-imagor](https://github.com/sandstorm/laravel-imagor)** provides seamless integration between Laravel applications and imagor.

**Features:**
- **Laravel service provider** for easy setup
- **Blade directives** for template integration
- **Configuration management** through Laravel config
- **URL generation helpers** for signed URLs
- **Middleware support** for request handling

**Installation:**
```bash
composer require sandstorm/laravel-imagor
```

**Usage:**
```php
// Generate signed imagor URL
$url = imagor()->url('path/to/image.jpg')
    ->resize(300, 200)
    ->quality(80)
    ->format('webp')
    ->get();

// In Blade templates
@imagor('image.jpg', ['width' => 300, 'height' => 200])
```

## Contributing to the Ecosystem

### Creating Integrations

When creating imagor integrations, consider these best practices:

1. **URL Signing**: Implement proper HMAC signing for security
2. **Error Handling**: Handle network errors and invalid responses
3. **Configuration**: Allow flexible configuration of imagor endpoints
4. **Documentation**: Provide clear examples and API documentation
5. **Testing**: Include comprehensive tests for different scenarios

### Sharing Your Work

If you've created an imagor integration or tool:

1. **Open Source**: Consider making it open source for community benefit
2. **Documentation**: Write clear README and usage examples
3. **Community**: Share in imagor discussions and forums
4. **Maintenance**: Keep integrations updated with imagor changes

### Getting Help

- **GitHub Issues**: Report bugs and request features
- **Discussions**: Join community discussions for help and ideas
- **Documentation**: Refer to official documentation and examples
- **Code Review**: Share your integrations for community feedback

## Community Guidelines

### Code of Conduct

- Be respectful and inclusive in all interactions
- Help others learn and grow in the community
- Share knowledge and best practices
- Provide constructive feedback and suggestions

### Contributing

- Follow established coding standards and conventions
- Write tests for new features and bug fixes
- Update documentation for any changes
- Submit pull requests with clear descriptions

The imagor community thrives on collaboration and shared knowledge. Whether you're building integrations, sharing examples, or helping others, your contributions make the ecosystem stronger for everyone.
