<?php

declare(strict_types=1);

namespace Linktor\Tests;

use Linktor\LinktorClient;
use Linktor\Resources\ChannelsResource;
use Linktor\Types\Channel;
use Linktor\Types\ConnectResult;
use PHPUnit\Framework\TestCase;

/**
 * The HttpClient already unwraps the `{ success, data }` envelope, so the
 * resource sees the inner `data`. These tests mock LinktorClient (whose public
 * get/post/put methods return that unwrapped `data`) to prove the resource
 * shapes it into the right wire types.
 */
class ChannelsResourceTest extends TestCase
{
    private function channelData(): array
    {
        return [
            'id' => 'ch_1',
            'tenant_id' => 't_1',
            'type' => 'whatsapp_unofficial',
            'name' => 'WhatsApp',
            'enabled' => true,
            'connection_status' => 'connecting',
            'config' => ['phone_number_id' => '123', 'access_token' => '__redacted__'],
            'created_at' => '2026-07-24T10:00:00Z',
            'updated_at' => '2026-07-24T10:00:00Z',
        ];
    }

    public function testConnectSurfacesQrCode(): void
    {
        // Inner `data` for POST /channels/{id}/connect, post-envelope-unwrap.
        $unwrapped = [
            'channel' => $this->channelData(),
            'qr_code' => 'Q',
            'expires_in' => 60,
        ];

        $client = $this->createMock(LinktorClient::class);
        $client->expects($this->once())
            ->method('post')
            ->with('/channels/ch_1/connect', [])
            ->willReturn($unwrapped);

        $channels = new ChannelsResource($client);
        $result = $channels->connect('ch_1');

        $this->assertInstanceOf(ConnectResult::class, $result);
        $this->assertSame('Q', $result->qrCode);
        $this->assertSame(60, $result->expiresIn);
        $this->assertInstanceOf(Channel::class, $result->channel);
        $this->assertSame('ch_1', $result->channel->id);
        $this->assertSame('connecting', $result->channel->connectionStatus);
    }

    public function testRequestPairCodeSendsPhoneAndSurfacesPairCode(): void
    {
        $unwrapped = [
            'channel' => $this->channelData(),
            'pair_code' => 'ABCD-1234',
            'expires_in' => 120,
        ];

        $client = $this->createMock(LinktorClient::class);
        $client->expects($this->once())
            ->method('post')
            ->with('/channels/ch_1/pair', ['phone_number' => '+15551234567'])
            ->willReturn($unwrapped);

        $channels = new ChannelsResource($client);
        $result = $channels->requestPairCode('ch_1', '+15551234567');

        $this->assertSame('ABCD-1234', $result->pairCode);
        $this->assertSame(120, $result->expiresIn);
    }

    public function testCreateSendsCredentials(): void
    {
        $input = [
            'type' => 'whatsapp',
            'name' => 'WhatsApp',
            'config' => ['phone_number_id' => '123'],
            'credentials' => ['access_token' => 'secret'],
        ];

        $client = $this->createMock(LinktorClient::class);
        $client->expects($this->once())
            ->method('post')
            ->with('/channels', $input)
            ->willReturn($this->channelData());

        $channels = new ChannelsResource($client);
        $channel = $channels->create($input);

        $this->assertInstanceOf(Channel::class, $channel);
        $this->assertSame('ch_1', $channel->id);
        $this->assertTrue($channel->enabled);
        // Credentials are write-only: they must never surface on the response.
        $this->assertArrayNotHasKey('access_token', $channel->config);
        $this->assertSame('__redacted__', $channel->config['access_token'] ?? null);
    }

    public function testUpdateUsesPut(): void
    {
        $client = $this->createMock(LinktorClient::class);
        $client->expects($this->once())
            ->method('put')
            ->with('/channels/ch_1', ['name' => 'Renamed'])
            ->willReturn(['id' => 'ch_1', 'tenant_id' => 't_1', 'type' => 'whatsapp', 'name' => 'Renamed']);

        $channels = new ChannelsResource($client);
        $channel = $channels->update('ch_1', ['name' => 'Renamed']);

        $this->assertSame('Renamed', $channel->name);
    }

    public function testListReturnsChannelArray(): void
    {
        $client = $this->createMock(LinktorClient::class);
        $client->expects($this->once())
            ->method('get')
            ->with('/channels?type=whatsapp')
            ->willReturn([$this->channelData(), $this->channelData()]);

        $channels = new ChannelsResource($client);
        $list = $channels->list(['type' => 'whatsapp']);

        $this->assertCount(2, $list);
        $this->assertContainsOnlyInstancesOf(Channel::class, $list);
    }
}
