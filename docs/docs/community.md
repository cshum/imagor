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

### imagor-toy

**[codedoga/imagor-toy](https://github.com/codedoga/imagor-toy)** is a ReactJS-based interactive playground for experimenting with imagor transformations.

**Features:**
- **Interactive interface** for testing transformations
- **Real-time preview** of image operations
- **URL generation** with copy functionality
- **Parameter exploration** for learning imagor
- **Educational tool** for developers

**Use Cases:**
- Learning imagor capabilities
- Testing transformation parameters
- Prototyping image workflows
- Developer education and training

## Integration Examples

### Node.js Integration

```javascript
const crypto = require('crypto');

class ImagorClient {
  constructor(baseUrl, secret) {
    this.baseUrl = baseUrl;
    this.secret = secret;
  }

  sign(path) {
    if (!this.secret) return `unsafe/${path}`;
    
    const hash = crypto.createHmac('sha1', this.secret)
      .update(path)
      .digest('base64')
      .replace(/\+/g, '-')
      .replace(/\//g, '_');
    
    return `${hash}/${path}`;
  }

  url(imagePath) {
    return {
      resize: (width, height) => ({ 
        ...this, 
        path: `${width}x${height}/${imagePath}` 
      }),
      quality: (q) => ({ 
        ...this, 
        path: `filters:quality(${q})/${this.path || imagePath}` 
      }),
      build: () => `${this.baseUrl}/${this.sign(this.path || imagePath)}`
    };
  }
}

// Usage
const imagor = new ImagorClient('http://localhost:8000', 'mysecret');
const url = imagor.url('image.jpg').resize(300, 200).quality(80).build();
```

### Python Integration

```python
import hmac
import hashlib
import base64
from urllib.parse import quote

class ImagorClient:
    def __init__(self, base_url, secret=None):
        self.base_url = base_url
        self.secret = secret
    
    def sign(self, path):
        if not self.secret:
            return f"unsafe/{path}"
        
        signature = hmac.new(
            self.secret.encode(),
            path.encode(),
            hashlib.sha1
        ).digest()
        
        signature_b64 = base64.b64encode(signature).decode()
        signature_url = signature_b64.replace('+', '-').replace('/', '_')
        
        return f"{signature_url}/{path}"
    
    def url(self, image_path):
        return ImagorURL(self, image_path)

class ImagorURL:
    def __init__(self, client, image_path):
        self.client = client
        self.image_path = image_path
        self.operations = []
    
    def resize(self, width, height):
        self.operations.append(f"{width}x{height}")
        return self
    
    def quality(self, q):
        self.operations.append(f"filters:quality({q})")
        return self
    
    def build(self):
        path = "/".join(self.operations + [self.image_path])
        signed_path = self.client.sign(path)
        return f"{self.client.base_url}/{signed_path}"

# Usage
imagor = ImagorClient('http://localhost:8000', 'mysecret')
url = imagor.url('image.jpg').resize(300, 200).quality(80).build()
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
