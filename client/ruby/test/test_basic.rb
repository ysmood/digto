# frozen_string_literal: true

require 'minitest/autorun'
require 'securerandom'
require 'http'
require 'digto'

class DigtoTest < MiniTest::Test
  def test_basic
    subdomain = SecureRandom.urlsafe_base64

    c = Digto::Client.new subdomain

    assert_equal subdomain, c.subdomain

    thr = Thread.new do
      res = `curl -s #{c.public_url}`
      assert_equal 'done', res
    end

    s = c.next

    s.response(200, {}, body: 'done')

    thr.join
  end
end
