# frozen_string_literal: true

Gem::Specification.new do |s|
  s.name        = 'digto'
  s.version     = '0.0.2'
  s.date        = '2019-12-24'
  s.summary     = 'Ruby implementaion for digto'
  s.description = 'https://github.com/ysmood/digto'
  s.authors     = ['Yad Smood']
  s.email       = 'ys@ysmood.org'
  s.files       = ['lib/digto.rb']
  s.homepage    = 'https://github.com/ysmood/digto'
  s.license = 'MIT'

  s.add_runtime_dependency 'http'

  s.add_development_dependency 'minitest'
end
